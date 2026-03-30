from logging import Logger
from pathlib import Path
import time
from typing import Any

from langchain.agents import create_agent
from langchain.agents.middleware import AgentMiddleware, ToolCallLimitMiddleware
from langchain.messages import HumanMessage, SystemMessage
from langchain_ollama import ChatOllama
from langchain.tools import BaseTool, tool
from langgraph.errors import GraphRecursionError
import ollama

from tools import agent_as_tool, WriteFileTool, ReadFilesTool
from utils import CustomLoggerCallback, Metrics, get_exception_text, clear_directory


PROMPTS_DIR = Path(__file__).parent / 'prompts'
NUM_RETRIES = 3


def create_agent_with_tools(
    model_name: str,
    base_url: str,
    prompt: str,
    tools: list[BaseTool],
    middleware: list[AgentMiddleware],
):
    model = ChatOllama(model=model_name, base_url=base_url)

    system_prompt = SystemMessage(
        content=[
            {
                'type': 'text',
                'text': prompt,
            },
        ],
    )

    return create_agent(
        model=model,
        system_prompt=system_prompt,
        tools=tools,
        middleware=middleware,
    )


def create_tools(
    model_name: str,
    ollama_base_url: str,
    logger: Logger,
    base_dir: Path,
    statement_path: Path,
    with_checker: bool = True,
) -> list[BaseTool]:
    read_files_tool = ReadFilesTool(logger, base_dir)
    write_file_tool = WriteFileTool(logger, base_dir, statement_path)
    tools = [
        read_files_tool,
        write_file_tool,
    ]

    if not with_checker:
        return tools

    with open(PROMPTS_DIR / 'validator.txt', 'r') as file:
        validator_system_prompt = file.read()

    validator = create_agent_with_tools(
        model_name=model_name,
        base_url=ollama_base_url,
        prompt=validator_system_prompt,
        tools = [read_files_tool],
        middleware=[
            ToolCallLimitMiddleware(thread_limit=50, run_limit=10),
            ToolCallLimitMiddleware(tool_name=read_files_tool.name, thread_limit=15, run_limit=10),
        ],
    )

    validator_tool = agent_as_tool(
        agent=validator,
        name='validate',
        description='Validate the generated code and tests for compliance with the user\'s request',
        message='validate code, tests and user input',
        logger=logger,
    )

    with open(PROMPTS_DIR / 'tester.txt', 'r') as file:
        tester_system_prompt = file.read()

    tester = create_agent_with_tools(
        model_name=model_name,
        base_url=ollama_base_url,
        prompt=tester_system_prompt,
        tools = [read_files_tool, write_file_tool, validator_tool],
        middleware=[
            ToolCallLimitMiddleware(thread_limit=50, run_limit=10),
            ToolCallLimitMiddleware(tool_name=read_files_tool.name, thread_limit=15, run_limit=10),
            ToolCallLimitMiddleware(tool_name=write_file_tool.name, thread_limit=15, run_limit=10),
            ToolCallLimitMiddleware(tool_name='validate', thread_limit=15, run_limit=10),
        ],
    )

    tester_tool = agent_as_tool(
        agent=tester,
        name='test',
        description='Test generated code',
        message='write tests for generated code',
        logger=logger,
    )

    with open(PROMPTS_DIR / 'checker.txt', 'r') as file:
        checker_system_prompt = file.read()

    checker = create_agent_with_tools(
        model_name=model_name,
        base_url=ollama_base_url,
        prompt=checker_system_prompt,
        tools = [read_files_tool, tester_tool],
        middleware=[
            ToolCallLimitMiddleware(thread_limit=50, run_limit=10),
            ToolCallLimitMiddleware(tool_name=read_files_tool.name, thread_limit=15, run_limit=10),
            ToolCallLimitMiddleware(tool_name='test', thread_limit=15, run_limit=10),
        ],
    )

    checker_tool = agent_as_tool(
        agent=checker,
        name='check',
        description='Сheck the generated code for compliance with the user\'s request',
        message='check the code and user input',
        logger=logger,
    )

    tools.append(checker_tool)
    return tools


def create_llm_coder(
    model_name: str,
    ollama_base_url: str,
    logger: Logger,
    base_dir: Path,
    statement_path: Path,
    with_checker: bool = True,
):
    tools = create_tools(model_name, ollama_base_url, logger, base_dir, statement_path, with_checker=with_checker)
    middleware = [
        ToolCallLimitMiddleware(tool_name=tool.name, thread_limit=15, run_limit=10)
        for tool in tools
    ]
    middleware.append(ToolCallLimitMiddleware(thread_limit=50, run_limit=10))

    prompt_filename = 'coder-single.txt'
    if with_checker:
        prompt_filename = 'coder.txt'

    with open(PROMPTS_DIR / prompt_filename, 'r') as file:
        coder_system_prompt = file.read()

    coder = create_agent_with_tools(
        model_name=model_name,
        base_url=ollama_base_url,
        prompt=coder_system_prompt,
        tools=tools,
        middleware=middleware,
    )

    return coder


class Coder:
    def __init__(
        self,
        model_name: str,
        ollama_base_url: str,
        logger: Logger,
        base_dir: Path,
        statement_path: Path,
        with_checker: bool = True,
    ):
        self._logger = logger
        self._base_dir = base_dir
        self._statement_path = statement_path

        self._coder = create_llm_coder(model_name, ollama_base_url, logger, base_dir, statement_path, with_checker=with_checker)

    def _process(self, query: str) -> Metrics:
        input_data = {
            'messages': [
                HumanMessage(query),
            ],
            'statement_path': self._statement_path,
            'base_dir': self._base_dir,
        }

        callback = CustomLoggerCallback(self._logger)

        now = time.perf_counter()
        try:
            self._coder.invoke(
                input_data,
                config={
                    'callbacks': [callback],
                },
            )
        except GraphRecursionError as exc:
            text = get_exception_text(exc, limit=1)
            self._logger.error(f'Coder invocation failed with GraphRecursionError:\n{text}')
            pass
        elapsed = time.perf_counter() - now

        metrics = callback.get_metrics()
        metrics |= {
            'total_time': elapsed,
        }

        return metrics

    def process(self, query: str) -> Metrics:
        for attempt in range(NUM_RETRIES):
            try:
                return self._process(query)
            except ollama._types.ResponseError as exc:
                self._logger.warn(f'Attempt #{attempt + 1} failed with status {exc.status_code}')
                if attempt + 1 == NUM_RETRIES:
                    raise
                clear_directory(self._base_dir, exclude=[self._statement_path])
