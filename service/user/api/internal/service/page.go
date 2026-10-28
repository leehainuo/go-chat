package service

import (
	"go-chat/service/user/api/internal/consts"
)

// ClampPage 分页参数修正：pageNum ≥ 1，pageSize ∈ [1, MaxPageSize]。
func ClampPage(pageNum, pageSize int64) (int64, int64) {
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > consts.MaxPageSize {
		pageSize = consts.MaxPageSize
	}
	return pageNum, pageSize
}

// PageSlice 内存分页：返回当前页切片（不越界，空页返回空切片）、
// 总条数、总页数与前后页标记。total 以过滤后的切片长度为准。
func PageSlice[T any](items []T, pageNum, pageSize int64) (pageItems []T, total, pages int64, hasNext, hasPrevious bool) {
	total = int64(len(items))
	if total == 0 {
		return []T{}, 0, 0, false, false
	}
	if pageSize < 1 {
		pageSize = 10
	}
	pages = (total + pageSize - 1) / pageSize
	start := (pageNum - 1) * pageSize
	if start >= total {
		return []T{}, total, pages, false, pageNum > 1
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return items[start:end], total, pages, pages > pageNum, pageNum > 1
}
