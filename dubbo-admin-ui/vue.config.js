const path = require('path');

module.exports = {
  outputDir: "target/dist",
  lintOnSave: "warning",
  devServer: {
    port: 8082,
    historyApiFallback: {
      rewrites: [
        {from: /.*/, to: path.posix.join('/', 'index.html')},
      ],
    },
    publicPath: '/',
    proxy: {
      '/': {
        target: 'http://localhost:8080/',
        changeOrigin: true,
        pathRewrite: {
          '^/': '/'
        }
      }
    }
  }
};
