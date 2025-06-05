// Copyright 2024 孔令飞 <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onexstack. The professional
// version of this repository is https://github.com/onexstack/onex.

package rid_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onexstack/onexstack/pkg/rid"
)

// Mock Salt function used for testing
func Salt() string {
	return "staticSalt"
}

func TestResourceID_String(t *testing.T) {
	// 测试 UserID 转换为字符串
	userID := rid.NewResourceID("user")
	assert.Equal(t, "user", userID.String(), "UserID.String() should return 'user'")

	// 测试 PostID 转换为字符串
	postID := rid.NewResourceID("post")
	assert.Equal(t, "post", postID.String(), "PostID.String() should return 'post'")
}

func TestResourceID_New(t *testing.T) {
	// 测试生成的ID是否带有正确前缀
	userID := rid.NewResourceID("user")
	uniqueID := userID.New(1)

	assert.True(t, len(uniqueID) > 0, "Generated ID should not be empty")
	assert.Contains(t, uniqueID, "user-", "Generated ID should start with 'user-' prefix")

	// 生成另外一个唯一标识符，确保生成的值不同（唯一性）
	anotherID := userID.New(2)
	assert.NotEqual(t, uniqueID, anotherID, "Generated IDs should be unique")
}

func BenchmarkResourceID_New(b *testing.B) {
	// 性能测试
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := rid.NewResourceID("user")
		_ = userID.New(uint64(i))
	}
}

func FuzzResourceID_New(f *testing.F) {
	// 添加预置测试数据
	f.Add(uint64(1))      // 添加一个种子值 counter 为 1
	f.Add(uint64(123456)) // 添加一个较大的种子值

	f.Fuzz(func(t *testing.T, counter uint64) {
		// 测试 ResourceID 的 New 方法
		result := rid.NewResourceID("user").New(counter)

		// 断言结果不为空
		assert.NotEmpty(t, result, "The generated unique identifier should not be empty")

		// 断言结果必须包含资源标识符前缀
		assert.Contains(t, result, rid.NewResourceID("user").String()+"-", "The generated unique identifier should contain the correct prefix")

		// 断言前缀不会与 uniqueStr 部分重叠
		splitParts := strings.SplitN(result, "-", 2)
		assert.Equal(t, rid.NewResourceID("user").String(), splitParts[0], "The prefix part of the result should correctly match the UserID")

		// 断言生成的 ID 具有固定长度（基于 NewCode 的配置）
		if len(splitParts) == 2 {
			assert.Equal(t, 6, len(splitParts[1]), "The unique identifier part should have a length of 6")
		} else {
			t.Errorf("The format of the generated unique identifier does not meet expectation")
		}
	})
}
