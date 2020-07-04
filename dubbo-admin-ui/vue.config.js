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
  },
  configureWebpack: {
    performance: {
      hints: false
    },
    optimization: {
      splitChunks: {
        cacheGroups: {
          reactBase: {
            name: 'braceBase',
            test: (module) => {
              return /brace/.test(module.context);
            },
            chunks: 'initial',
            priority: 10,
          },
          common: {
            name: 'vendor',
            chunks: 'initial',
            priority: 2,
            minChunks: 2,
          },
        }
      }
    }
  }
};
