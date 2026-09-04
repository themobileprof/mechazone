#!/usr/bin/env bash
# Install BAAI/bge-small-en-v1.5 (~37MB Q8) into local Ollama. Fits a 2GB box.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
dir="$root/data/models"
mkdir -p "$dir"
gguf="$dir/bge-small-en-v1.5-q8_0.gguf"
url="https://huggingface.co/CompendiumLabs/bge-small-en-v1.5-gguf/resolve/main/bge-small-en-v1.5-q8_0.gguf"
if [[ ! -s "$gguf" ]]; then
	echo "downloading bge-small-en-v1.5 Q8 GGUF (~37MB)"
	curl -4 -L --fail -o "$gguf" "$url"
fi
if ! command -v ollama >/dev/null; then
	echo "install Ollama first: https://ollama.com/download" >&2
	exit 1
fi
if ! ollama list 2>/dev/null | grep -q 'bge-small-en-v1.5'; then
	ollama create bge-small-en-v1.5 -f "$dir/Modelfile.bge-small-en-v1.5"
fi
echo "bge-small-en-v1.5 ready (384-dim, ~37MB). Then: make embed"
