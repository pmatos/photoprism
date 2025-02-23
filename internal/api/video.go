package api

import (
	"net/http"

	"github.com/photoprism/photoprism/pkg/sanitize"

	"github.com/photoprism/photoprism/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/photoprism/photoprism/internal/photoprism"
	"github.com/photoprism/photoprism/internal/query"
	"github.com/photoprism/photoprism/internal/video"
)

// GetVideo streams videos.
//
// GET /api/v1/videos/:hash/:token/:type
//
// Parameters:
//   hash: string The photo or video file hash as returned by the search API
//   type: string Video type
func GetVideo(router *gin.RouterGroup) {
	router.GET("/videos/:hash/:token/:type", func(c *gin.Context) {
		if InvalidPreviewToken(c) {
			c.Data(http.StatusForbidden, "image/svg+xml", brokenIconSvg)
			return
		}

		fileHash := sanitize.Token(c.Param("hash"))
		typeName := sanitize.Token(c.Param("type"))

		videoType, ok := video.Types[typeName]

		if !ok {
			log.Errorf("video: invalid type %s", sanitize.Log(typeName))
			c.Data(http.StatusOK, "image/svg+xml", videoIconSvg)
			return
		}

		f, err := query.FileByHash(fileHash)

		if err != nil {
			log.Errorf("video: %s", err.Error())
			c.Data(http.StatusOK, "image/svg+xml", videoIconSvg)
			return
		}

		if !f.FileVideo {
			f, err = query.VideoByPhotoUID(f.PhotoUID)

			if err != nil {
				log.Errorf("video: %s", err.Error())
				c.Data(http.StatusOK, "image/svg+xml", videoIconSvg)
				return
			}
		}

		if f.FileError != "" {
			log.Errorf("video: file error %s", f.FileError)
			c.Data(http.StatusOK, "image/svg+xml", videoIconSvg)
			return
		}

		fileName := photoprism.FileName(f.FileRoot, f.FileName)

		if mf, err := photoprism.NewMediaFile(fileName); err != nil {
			log.Errorf("video: file %s is missing", sanitize.Log(f.FileName))
			c.Data(http.StatusOK, "image/svg+xml", videoIconSvg)

			// Set missing flag so that the file doesn't show up in search results anymore.
			logError("video", f.Update("FileMissing", true))

			return
		} else if f.FileCodec != string(videoType.Codec) {
			conv := service.Convert()

			if avcFile, err := conv.ToAvc(mf, service.Config().FFmpegEncoder()); err != nil {
				log.Errorf("video: transcoding %s failed", sanitize.Log(f.FileName))
				c.Data(http.StatusOK, "image/svg+xml", videoIconSvg)
				return
			} else {
				fileName = avcFile.FileName()
			}
		}

		AddContentTypeHeader(c, ContentTypeAvc)

		if c.Query("download") != "" {
			c.FileAttachment(fileName, f.DownloadName(DownloadName(c), 0))
		} else {
			c.File(fileName)
		}

		return
	})
}
