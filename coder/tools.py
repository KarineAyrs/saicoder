from logging import Logger
from pathlib import Path
from typing import Type

from langchain.tools import BaseTool, tool
from pydantic import BaseModel, Field

from utils import is_path_in_subdirectory, parse_markdown_tag


class WriteFileInput(BaseModel):
    filepath: str = Field(..., description='Path to the file to be written')
    text: str = Field(..., description='Chat model text')
    tag: str = Field(..., description='Tag to parse from Markdown')


class WriteFileTool(BaseTool):
    args_schema: Type[BaseModel] = WriteFileInput
    name: str = 'write_file'
    description: str = 'Write to a text file.'

    def __init__(self, logger: Logger, base_dir: Path, statement_path: Path, **kwargs):
        super().__init__(**kwargs)

        self._logger = logger
        self._base_dir = base_dir
        self._statement_path = statement_path

    def _run(self, filepath: str, text: str, tag: str) -> None:
        self._logger.debug(f'write_file tool called for path {filepath}')

        path = self._base_dir / filepath
        if not is_path_in_subdirectory(self._base_dir, path):
            raise ValueError(f'Filepath {filepath} should be a subpath of {self._base_dir}')

        if path == self._statement_path:
            raise ValueError(f'Cannot overwrite problem statement')

        parent_dir = path.parent
        parent_dir.mkdir(parents=True, exist_ok=True)

        result = parse_markdown_tag(text, tag)
        with open(path, 'w') as file:
            file.write(result)


class ReadFilesInput(BaseModel):
    pass


class ReadFilesTool(BaseTool):
    args_schema: Type[BaseModel] = ReadFilesInput
    name: str = 'read_files'
    description: str = 'Read all text files.'

    def __init__(self, logger: Logger, base_dir: Path, **kwargs):
        super().__init__(**kwargs)

        self._logger = logger
        self._base_dir = base_dir

    def _run(self) -> list[str]:
        self._logger.debug(f'read_files tool called')

        result = []
        for path in self._base_dir.rglob('*'):
            if any(part.startswith('.') for part in path.parts) or not path.is_file():
                continue

            try:
                content = path.read_text(encoding='utf-8', errors='ignore')
            except Exception as exc:
                content = f'<<ERROR READING FILE: {exc}>>'

            result.append(f'>>> {path}\n{content}')

        return result


def agent_as_tool(
    agent,
    name: str,
    description: str,
    message: str,
    logger: Logger,
) -> BaseTool:
    @tool(name, description=description)
    def call_agent():
        logger.debug(f'{name} tool called')

        result = agent.invoke({
            'messages': [
                {
                    'role': 'user',
                    'content': message,
                },
            ],
        })

        return result['messages'][-1].content

    return call_agent
