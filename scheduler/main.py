#!/usr/bin/env python3

from io import StringIO
import logging
import traceback

from src.bin import cli  # type: ignore


def main():
    try:
        cli()
    except Exception as exc:
        buffer = StringIO()
        traceback.print_exception(exc, limit=-3, file=buffer)
        logging.critical(f'Exception occurred during scheduler run:\n{buffer.getvalue()}\n\n')


if __name__ == '__main__':
    main()
