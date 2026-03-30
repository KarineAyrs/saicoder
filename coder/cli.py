import json
import logging
from logging import Logger
from pathlib import Path
import sys

import click

from agents import Coder
from utils import get_logger, get_exception_text


@click.command()
@click.option(
    '--log',
    '-l',
    'log_path',
    type=click.Path(dir_okay=False, file_okay=True, writable=True, resolve_path=True, allow_dash=True, path_type=Path),  # type: ignore
    default='-',
    show_default=True,
    help='Path to log file (pass "-" to log to stderr)',
)
@click.option(
    '--workdir',
    '-w',
    'workdir',
    type=click.Path(dir_okay=True, file_okay=False, writable=True, resolve_path=True, path_type=Path),  # type: ignore
    default='.',
    show_default=True,
    help='Path to working directory',
)
@click.option(
    '--statement',
    '-s',
    'statement_relpath',
    type=click.Path(dir_okay=False, file_okay=True, readable=True, path_type=Path),  # type: ignore
    default='.',
    show_default=True,
    help='Relative path to statement file',
)
@click.option(
    '--model',
    '-m',
    'model_name',
    type=str,
    default='qwen3-coder:480b-cloud',
    show_default=True,
    help='Model name',
)
@click.option(
    '--ollama-base-url',
    'ollama_base_url',
    type=str,
    default='http://ollama:11434',
    show_default=True,
    help='Ollama base URL',
)
@click.option(
    '--enable-checker/--disable-checker',
    'with_checker',
    is_flag=True,
    default=True,
    show_default=True,
    help='Enable multi-agent system with code checker',
)
def cli(
    log_path: Path,
    workdir: Path,
    statement_relpath: Path,
    model_name: str,
    ollama_base_url: str,
    with_checker: bool,
) -> None:
    '''LLM-coder'''

    logging.basicConfig()
    logger = get_logger('coder-logger', log_path, 'DEBUG')
    logger.debug(f'(logger) Workdir {workdir}, statement relpath {statement_relpath}')

    statement_path = workdir / statement_relpath
    coder = Coder(model_name, ollama_base_url, logger, workdir, statement_path, with_checker=with_checker)

    query = f'''\
Problem statement is written to a file {statement_path}.
Produce all the necessary code to solve the problem.
Do not try to overwrite the problem statement.'''

    try:
        metrics = coder.process(query)
        print(json.dumps(metrics, indent=4))
    except Exception as exc:
        text = get_exception_text(exc, limit=3)
        logger.critical(f'Exception occurred during query processing:\n{text}\n\n')
        sys.exit(1)
