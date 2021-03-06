package etl

import (
	"strings"
	"testing"
	"time"

	"github.com/livepeer/cdn-log-puller/internal/utils"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

var testLines = `
2021-11-17	16:47:16	GET	104.28.131.0	https	https://cdn.livepeer.monster/	Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.1 Safari/605.1.15	0	736	0	151.139.34.203	2.147	499	msn=516&mTrack=1&dur=2000	/hls/video+9e70xehvtu637q6p/5/chunk_1031999.ts	-	-
2021-11-17	16:47:17	GET	104.28.131.0	https	https://cdn.livepeer.monster/	Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.1 Safari/605.1.15	72756	736	74134	151.139.34.203	0.542	200	msn=516&mTrack=1&dur=2000	/hls/video+9e70xehvtu637q6p/5/chunk_1031999.ts	-	-
2021-11-17	16:47:17	GET	104.28.131.0	https	https://cdn.livepeer.monster/	Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.1 Safari/605.1.15	81780	736	83205	151.139.34.195	0.784	200	msn=517&mTrack=1&dur=2000	/hls/video+9e70xehvtu637q6p/5/chunk_1033999.ts	-	-
2021-11-17	20:49:04	GET	104.28.106.0	https	https://cdn.livepeer.monster/cmaf/9e70xehvtu637q6p/index.m3u8	Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.1 Safari/605.1.15	18584	777	20029	151.139.86.3	0.186	200	msn=5706&mTrack=1&dur=500&sessId=3405774711	/cmaf/9e70xehvtu637q6p/5/chunk_11411999.2.m4s	-	-
2021-11-17	20:43:56	GET	104.28.106.0	https	https://cdn.livepeer.monster/cmaf/9e70xehvtu637q6p/index.m3u8	Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.1 Safari/605.1.15	10484	756	11929	151.139.86.19	0.260	200	msn=345&mTrack=1&dur=500	/cmaf/9e70xehvtu637q6p/0/chunk_689999.0.m4s	-	-
`

func TestParseLine(t *testing.T) {
	assert := assert.New(t)
	assert.True(true)
	lines := strings.Split(testLines, "\n")
	lines = lines[1:]
	datac := make(chan VideoStat, len(lines))
	var vs VideoStat
	err := parseLine(lines[0], datac)
	if !assert.NoError(err) {
		return
	}
	vs = <-datac
	assert.Equal("9e70xehvtu637q6p", vs.streamId)
	assert.Equal("499", vs.httpCode)
	assert.Equal(utils.IDTypeManifestID, vs.itemType)
	assert.Equal(int64(0), vs.ScBytes)
	assert.Equal("2021-11-1716", vs.date)
	err = parseLine(lines[1], datac)
	if !assert.NoError(err) {
		return
	}
	vs = <-datac
	assert.Equal("9e70xehvtu637q6p", vs.streamId)
	assert.Equal("200", vs.httpCode)
	assert.Equal(utils.IDTypeManifestID, vs.itemType)
	assert.Equal(int64(74134), vs.ScBytes)
	assert.Equal("2021-11-1716", vs.date)
	err = parseLine(lines[3], datac)
	if !assert.NoError(err) {
		return
	}
	vs = <-datac
	assert.Equal("9e70xehvtu637q6p", vs.streamId)
	assert.Equal("200", vs.httpCode)
	assert.Equal(utils.IDTypeManifestID, vs.itemType)
	assert.Equal(int64(20029), vs.ScBytes)
	assert.Equal("2021-11-1720", vs.date)
}

func TestAggregation(t *testing.T) {
	assert := assert.New(t)
	assert.True(true)
	lines := strings.Split(testLines, "\n")
	datac := make(chan VideoStat, len(lines))
	ctx, cancel := context.WithCancel(context.Background())
	agg := newAggregator(ctx, nil, "bucket", "", "")

	doneChan := make(chan struct{})

	go agg.incomingDataLoop(doneChan, datac)
	for _, line := range lines {
		if line == "" {
			continue
		}
		err := parseLine(line, datac)
		if !assert.NoError(err) {
			return
		}
	}
	close(datac)
	<-doneChan
	res := agg.flatten("test-region", time.Now(), "test.file.name")
	assert.Len(res, 2)
	res1 := res[0]
	assert.Equal("test-region", res1.Region)
	assert.Equal("test.file.name", res1.FileName)
	assert.Equal(int64(1637164800), res1.Date)
	assert.Len(res1.Data, 1)
	d1 := res1.Data[0]
	assert.Equal(3, d1.Count)
	assert.Equal("", d1.StreamID)
	assert.Equal("9e70xehvtu637q6p", d1.PlaybackID)
	assert.Equal(int64(736*3), d1.TotalCsBytes)
	assert.Equal(int64(74134+83205), d1.TotalScBytes)
	assert.Equal(int64(72756+81780), d1.TotalFilesize)
	assert.Equal(1, d1.UniqueUsers)
	res1 = res[1]
	assert.Equal("test-region", res1.Region)
	assert.Equal("test.file.name", res1.FileName)
	assert.Equal(int64(1637179200), res1.Date)
	assert.Len(res1.Data, 1)
	d1 = res1.Data[0]
	assert.Equal(2, d1.Count)
	assert.Equal("", d1.StreamID)
	assert.Equal("9e70xehvtu637q6p", d1.PlaybackID)
	assert.Equal(int64(777+756), d1.TotalCsBytes)
	assert.Equal(int64(20029+11929), d1.TotalScBytes)
	assert.Equal(int64(18584+10484), d1.TotalFilesize)
	assert.Equal(1, d1.UniqueUsers)

	cancel()
}
