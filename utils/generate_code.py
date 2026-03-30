#!/usr/bin/python3

import base64
import json
from pathlib import Path
import subprocess
import time
from typing import Any
import uuid

import click
import requests
from tqdm import tqdm


USER_ID = '00000000-0000-0000-0000-000012345678'
DEFAULT_POLLING_TIMEOUT = 5  # seconds
ROOT_DIR = Path(__file__).resolve().parent.parent

Task = dict[str, Any]
Dataset = list[Task]


def read_dataset(path: Path) -> Dataset:
    with open(path, 'r') as file:
        dataset = [json.loads(line) for line in file]
    return dataset


def submit_task(task: Task) -> str:
    task_id = task['task_id']
    idempotency_key = f'00000000-0000-0000-0000-00000000{task_id:04d}'

    resp = requests.post(
        'http://localhost:8080/submit',
        json={
            'user_id': USER_ID,
            'statement': task['prompt'],
        },
        headers={
            'Idempotency-Key': idempotency_key,
        },
    )
    resp.raise_for_status()

    return resp.json()['task_id']


def get_task_result(task_id: str) -> tuple[str | None, bool]:
    resp = requests.get(
        f'http://localhost:8080/task/{task_id}',
        json={},
    )
    resp.raise_for_status()

    data = resp.json()
    status = data['status']
    if status == 'PROCESSING':
        return None, False
    elif status == 'DONE':
        return data['result'], True
    elif status == 'FAILED':
        return None, True

    raise ValueError(f'WTF with this task? {data}')


def run_cmd_with_json(cmd: list[str]) -> Any:
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise ValueError(f'Command {cmd} exited with status {result.returncode}: {result.stderr}')
    return json.loads(result.stdout)


def get_task_stats(task_id: str) -> str | None:
    # TODO: add http handler to worker
    return run_cmd_with_json(['docker', 'exec', '-it', 'worker', 'cat', f'/app/worker/stats/{task_id}.json'])


def process_task(task: Task, output_dir: Path, stats_dir: Path, polling_timeout: float = DEFAULT_POLLING_TIMEOUT) -> None:
    idx = task['task_id']
    stats_path = stats_dir / f'{idx:03d}.json'
    code_path = output_dir / f'{idx:03d}.zip'

    if stats_path.exists() and code_path.exists():
        return

    task_id = submit_task(task)
    while True:
        result, finished = get_task_result(task_id)
        if finished:
            if result is not None:
                break
            return
        time.sleep(polling_timeout)

    stats = get_task_stats(task_id)
    if stats.get('tool_calls', {}).get('write_file', 0) == 0:
        # do not store results if zip contains only statement.md
        return

    with open(stats_path, 'w') as file:
        print(json.dumps(stats, indent=4), file=file)

    output = base64.b64decode(result)
    with open(code_path, 'wb') as file:
        file.write(output)


@click.command()
@click.option(
    '--output-dir',
    '-o',
    'output_dir',
    type=click.Path(dir_okay=True, file_okay=False, writable=True, resolve_path=True, path_type=Path),
    required=True,
    help='Path to output directory',
)
@click.option(
    '--stats-dir',
    '-s',
    'stats_dir',
    type=click.Path(dir_okay=True, file_okay=False, writable=True, resolve_path=True, path_type=Path),
    required=True,
    help='Path to stats directory',
)
@click.option(
    '--from',
    'from_task',
    type=int,
    default=1,
    show_default=True,
    help='First task to continue processing from',
)
@click.option(
    '--to',
    'to_task',
    type=int,
    default=None,
    show_default=True,
    help='Last task to process',
)
@click.option(
    '--polling-timeout',
    '-t',
    'polling_timeout',
    type=float,
    default=5,
    show_default=True,
    help='Task polling timeout (in seconds)',
)
def main(
    output_dir: Path,
    stats_dir: Path,
    from_task: int,
    to_task: int | None,
    polling_timeout: float,
) -> None:
    dataset = read_dataset(ROOT_DIR / 'dataset.jsonl')
    if to_task is None:
        to_task = len(dataset)

    output_dir.mkdir(parents=True, exist_ok=True)
    stats_dir.mkdir(parents=True, exist_ok=True)
    for task in tqdm(dataset, desc='Dataset processing'):
        time.sleep(0.01)
        task_id = task['task_id']
        if task_id < from_task or task_id > to_task:
            continue
        process_task(task, output_dir, stats_dir, polling_timeout=polling_timeout)


if __name__ == '__main__':
    main()
