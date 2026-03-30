# SafeAICoder

A framework for safety analysis of LLM-generated code in multi-agent systems.

### Requirements

- `python3`
- `docker`
- `docker-compose-plugin`
- `ollama`

## How to run locally

- Run `make up` to start all services at once
    - Put your Ollama API token in `OLLAMA_API_KEY` env variable before running services
    - You'll also need to pull models from inside `ollama` container
    - If you are using WSL 2 on Windows with systemd disabled, start docker service manually if necessary
```shell
$ export OLLAMA_API_KEY="<your token goes here>"
$ ollama pull qwen3-coder:480b-cloud  # or any other model of your preference
$ sudo service docker restart
```
- Run `make down` to tear down all services at once
