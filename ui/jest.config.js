/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

module.exports = {
  preset: "@vue/cli-plugin-unit-jest",
  verbose: true, // 多于一个测试文件运行时展示每个测试用例测试通过情况
  bail: true, // 参数指定只要有一个测试用例没有通过，就停止执行后面的测试用例
  testEnvironment: 'jsdom', // 测试环境，jsdom可以在Node虚拟浏览器环境运行测试
  moduleFileExtensions: [ // 需要检测测的文件类型
    'js',
    'jsx',
    'json',
    // tell Jest to handle *.vue files
    'vue'
  ],
  transform: { // 预处理器配置，匹配的文件要经过转译才能被识别，否则会报错
    '.+\\.(css|styl|less|sass|scss|jpg|jpeg|png|svg|gif|eot|otf|webp|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga|avif)$':
    require.resolve('jest-transform-stub'),
    '^.+\\.jsx?$': require.resolve('babel-jest')
  },
  transformIgnorePatterns: ['/node_modules/'], // 转译时忽略 node_modules
  moduleNameMapper: { // 从正则表达式到模块名称的映射，和webpack的alisa类似
    '^@/(.*)$': '<rootDir>/src/$1'
  },
  snapshotSerializers: [ // Jest在快照测试中使用的快照序列化程序模块的路径列表
    'jest-serializer-vue'
  ],
  testMatch: [ // Jest用于检测测试的文件，可以用正则去匹配
    '**/tests/unit/**/*.spec.[jt]s?(x)',
    '**/__tests__/*.[jt]s?(x)'
  ],
  collectCoverage: true, // 覆盖率报告，运行测试命令后终端会展示报告结果
  collectCoverageFrom: [ // 需要进行收集覆盖率的文件，会依次进行执行符合的文件
    'src/**/*.{js,vue}',
  ],
  coverageReporters: ['html', 'lcov', 'text-summary'], // Jest在编写覆盖率报告的配置，添加"text"或"text-summary"在控制台输出中查看覆盖率摘要
  coveragePathIgnorePatterns: ['/node_modules/'], // 需要跳过覆盖率信息收集的文件目录
  coverageDirectory: "<rootDir>/tests/unit/coverage", // Jest输出覆盖信息文件的目录，运行测试命令会自动生成如下路径的coverage文件
  coverageThreshold: { // 覆盖结果的最低阈值设置，如果未达到阈值，jest将返回失败
    global: {
      branches: 60,
      functions: 80,
      lines: 80,
      statements: 80,
    },
  },
  testURL: 'http://localhost/', // 官网没有解释，经过尝试可以随意配置ip，如果识别不出是ip则会报错
  watchPlugins: [ // jest监视插件
    require.resolve('jest-watch-typeahead/filename'),
    require.resolve('jest-watch-typeahead/testname')
  ]
};
