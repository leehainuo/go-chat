package user

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"go-chat/pkg/result"
	"go-chat/pkg/utils"
	"go-chat/service/user/api/internal/consts"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"
)

var allowedImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

type UploadUrlLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadUrlLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadUrlLogic {
	return &UploadUrlLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UploadUrl 签发 MinIO 预签名 PUT URL：客户端直传对象存储，不经后端中转，不暴露 AK/SK。
// 预签名有效期 5 分钟（写权限泄漏窗口控制），文件名服务端生成，仅允许图片后缀。
func (l *UploadUrlLogic) UploadUrl(req *types.UploadUrlReq) (*types.UploadUrlResp, error) {
	userId, err := mustUserId(l.ctx)
	if err != nil {
		return nil, err
	}
	if l.svcCtx.Minio == nil {
		return nil, result.System("对象存储不可用，请联系管理员", nil)
	}
	var ext string
	if fileName := strings.TrimSpace(req.FileName); fileName != "" {
		ext = strings.ToLower(filepath.Ext(fileName))
		if !allowedImageExts[ext] {
			return nil, result.Param("仅支持 jpg/jpeg/png/gif/webp 格式", nil)
		}
	} else {
		ext = ".png"
	}
	objectId, err := utils.NextId()
	if err != nil {
		return nil, result.System("生成文件名失败", err)
	}
	objectKey := fmt.Sprintf("%s/%d/%d%s", consts.AvatarObjectPrefix, userId, objectId, ext)

	uploadUrl, err := l.svcCtx.Minio.PresignedPutObject(l.ctx,
		l.svcCtx.Config.Minio.Bucket, objectKey,
		time.Duration(consts.PresignTTLSeconds)*time.Second)
	if err != nil {
		return nil, result.System("生成预签名 URL 失败", err)
	}

	scheme := "https"
	if !l.svcCtx.Config.Minio.Secure {
		scheme = "http"
	}
	downloadUrl := fmt.Sprintf("%s://%s/%s/%s",
		scheme, l.svcCtx.Config.Minio.Endpoint, l.svcCtx.Config.Minio.Bucket, objectKey)

	return &types.UploadUrlResp{
		UploadUrl:   uploadUrl.String(),
		DownloadUrl: downloadUrl,
		ObjectKey:   objectKey,
	}, nil
}
