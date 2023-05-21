// 1、引入mockjs
const Mock = require('mockjs')

// 2、获取 mock.Random 对象
const random = Mock.Random
console.log(random) // 简单使用就不操作了，需要操作的可以去看文档

// 3、基本用法 Mock.mock(url, type, data) // 参数文档 https://github.com/nuysoft/Mock/wiki
Mock.mock('/mock/user/list', 'get', {
  code: 200,
  message: '成功',
  data: {
    // 生成十个如下格式的数据
    'list|10': [
      {
        'id|+1': 1, // 数字从当前数开始依次 +1
        'age|18-40': 20, // 年龄为18-40之间的随机数字
        'sex|1': ['男', '女'], // 性别是数组中随机的一个
        name: '@cname', // 名字为随机中文名字
        email: '@email', // 随机邮箱
        isShow: '@boolean' // 随机获取boolean值
      }
    ]
  }
})
