#!/usr/bin/env python3

import logging
import sys

from cli import cli
from utils import get_exception_text


def main():
    try:
        cli()
    except Exception as exc:
        text = get_exception_text(exc, limit=None)
        logging.critical(f'Exception occurred during coder run:\n{text}\n\n')
        sys.exit(1)


if __name__ == '__main__':
    main()
