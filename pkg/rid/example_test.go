// Copyright 2024 孔令飞 <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onexstack. The professional
// version of this repository is https://github.com/onexstack/onex.

package rid_test

import (
	"fmt"

	"github.com/onexstack/onexstack/pkg/rid"
)

func ExampleResourceID_String() {
	// 定义一个资源标识符，例如用户资源
	userID := rid.NewResourceID("user")

	// 调用 String 方法，将 ResourceID 类型转换为字符串类型
	idString := userID.String()

	// 输出结果
	fmt.Println(idString)

	// Output:
	// user
}
