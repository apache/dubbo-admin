# dubbo-admin-ai
## Introduction

This project is an intelligent agent's server for dubbo-admin.


## Startup
1. Set your API Keys in the `.env` file or set them as environment variables
```shell
# .env_example
PINECONE_API_KEY=your_pinecone_api_key
DASHSCOPE_API_KEY=your_dashscope_api_key
COHERE_API_KEY=your_cohere_api_key
SILICONFLOW_API_KEY=your_siliconflow_api_key
GEMINI_API_KEY=your_gemini_api_key
```

2. Run the server
```shell
go run main.go --mode dev --env your_env_file_path --port 8888
```

## Build and Run
```shell
mkdir build
cd build
go build -o dubbo-admin-ai ../

./dubbo-admin-ai --mode prod --env your_env_file_path --port 8888
```