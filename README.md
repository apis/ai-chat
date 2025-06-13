# AI Chat Project

#### TODO items

* Tools integration
* Sessions type locking
* Session timeout and scheduled cleanup
* Websocket behavior after time out
* Incremental update from server
* Disable controls when server is answering
* Cancel answer button
* "Answering" UI spinner
* Format user input as text - preserve new lines / spaces


curl -LsSf https://astral.sh/uv/install.sh | sh

mcphost -m ollama:granite3.3:8b --config "./configs/mcphost.config.json"

./mcphost -m "ollama:qwen3:8b" --config "../ai-chat/configs/mcphost.config.json"