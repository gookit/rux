package util

import (
	"fmt"
	"strings"
)

// Panicf 格式化输出并触发 panic
func Panicf(format string, a ...any) {
	panic(fmt.Sprintf(format, a...))
}

// ValidateOptionalSegments 验证可选参数规则
//
// 规则：
// 1. 可选参数只能在路径最后
// 2. 只能支持一个可选参数
func ValidateOptionalSegments(path string) {
	firstOptionalPos := strings.IndexByte(path, '[')
	lastOptionalPos := strings.LastIndexByte(path, '[')

	// 没有可选参数，直接返回
	if firstOptionalPos == -1 {
		return
	}

	// 规则 1：不能有多个可选参数
	if firstOptionalPos != lastOptionalPos {
		Panicf("route %s: only one optional segment is allowed", path)
	}

	// 规则 2：可选参数后不能有其他路径段
	closingBracketPos := strings.IndexByte(path, ']')
	afterOptionalPos := closingBracketPos + 1
	if afterOptionalPos < len(path) {
		Panicf("route %s: optional segment must be at the end of the path, found '%s' after ']'",
			path, path[afterOptionalPos:])
	}
}
