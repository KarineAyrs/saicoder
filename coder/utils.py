from collections import Counter, defaultdict
from io import StringIO
import logging
from pathlib import Path
import sys
import time
import traceback
from typing import Any

from langchain_core.callbacks import BaseCallbackHandler


LOG_FORMAT = '%(asctime)s - [%(levelname)s] - %(message)s'

Metrics = dict[str, Any]


def get_handler(path: Path, level: str) -> logging.Handler:
    handler: logging.Handler | None = None
    if path == '-':
        handler = logging.StreamHandler(sys.stderr)
    else:
        handler = logging.FileHandler(path)

    handler.setLevel(level)
    handler.setFormatter(logging.Formatter(LOG_FORMAT))
    return handler


def get_logger(name: str, path: Path, level: str) -> logging.Logger:
    logger = logging.getLogger(name)
    logger.setLevel(level)
    handler = get_handler(path, level)
    logger.addHandler(handler)
    return logger


def get_exception_text(exc: Exception, limit: int | None = None) -> str:
    buffer = StringIO()
    traceback.print_exception(exc, limit=limit, file=buffer)
    return buffer.getvalue()


def truncate_text(text: str, max_len: int) -> str:
    text = text.replace('\n', ' ')
    if len(text) > max_len:
        return text[:max_len] + '<...truncated>'
    return text


class CustomLoggerCallback(BaseCallbackHandler):
    def __init__(self, logger: logging.Logger, max_chars=100):
        super().__init__()

        self._logger = logger
        self._max_chars = max_chars

        self._tool_calls = Counter()
        self._tool_durations = defaultdict(float)
        self._tool_call_time = defaultdict(list)

    def on_llm_start(self, serialized, prompts, **kwargs):
        prompt = truncate_text(prompts[0], self._max_chars)
        self._logger.debug(f'LLM START: prompt={prompt}...')

    def on_llm_end(self, response, **kwargs):
        if response.llm_output is None:
            return
        output = truncate_text(response.llm_output['content'], self._max_chars)
        self._logger.debug(f'LLM END: response={output}...')

    def on_tool_start(self, serialized, input_str, **kwargs):
        name = serialized['name']
        input_str = truncate_text(input_str, self._max_chars)
        self._logger.debug(f'TOOL START: {name} input={input_str}')

        now = time.perf_counter()
        self._tool_calls[name] += 1
        self._tool_call_time[name].append(now)

    def on_tool_end(self, output, **kwargs):
        name = kwargs['name']
        output = truncate_text(output.text, self._max_chars)
        self._logger.debug(f'TOOL END: {name} output={output}')

        now = time.perf_counter()
        last_time = self._tool_call_time[name][-1]
        self._tool_durations[name] += now - last_time

    def get_metrics(self) -> Metrics:
        return {
            'tool_calls': self._tool_calls,
            'tool_durations': self._tool_durations,
        }


def parse_markdown_tag(markdown: str, tag: str) -> str:
    prefix = f'```{tag}\n'
    begin = markdown.rfind(prefix)
    if begin == -1:
        return markdown
    begin += len(prefix)

    end = markdown[begin:].find('```')
    if end == -1:
        end = len(markdown)
    else:
        end += begin

    return markdown[begin:end]


def is_path_in_subdirectory(directory: str | Path, path: str | Path) -> bool:
    directory = Path(directory).resolve()
    path = Path(path).resolve()

    return str(path).startswith(str(directory))


def clear_directory(directory: Path, exclude: list[Path] | None = None) -> None:
    if not directory.is_dir():
        return

    if not exclude:
        exclude = []
    protected = {
        directory / rel
        for rel in exclude
        if (directory / rel).exists()
    }

    def should_delete(path: Path) -> bool:
        return not any(
            protected_path in path.parents or path == protected_path
            for protected_path in protected
        )

    for child in directory.iterdir():
        if child.is_file():
            if should_delete(child):
                child.unlink()
        elif child.is_dir():
            if should_delete(child):
                child_exclude = [
                    prot.relative_to(child)
                    for prot in protected
                    if prot.is_relative_to(child)
                ]
                clear_directory(child, child_exclude)

            try:
                child.rmdir()
            except OSError:
                pass
