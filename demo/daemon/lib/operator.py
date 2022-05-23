import json
import subprocess as sp
import time
from typing import Callable

from apscheduler.schedulers.background import BackgroundScheduler
from loguru import logger

from lib.db import InfluxDb


def nibi_call(args: list[str]):
    """
    Call nibi cli with arguments

    Args:
        args (list[str]): _description_
    """
    default_args = ["--output=json"]
    return json.loads(sp.getoutput(" ".join(["nibid"] + args + default_args)))


class Operation:
    def __init__(
        self,
        name: str,
        table_name: str,
        args: list[str] = [],
        tags: list[str] = [],
        interval: float = 1,
        etl_fn: Callable[[dict], dict] = None,
    ):
        """
        Create an operation object to query from Nibiru CLI.

        Args:
            name (str): Name of the operator
            args (list[str]): List of arguments sent to the CLI
            tags (list[str]): List of tags for the influx push
            interval (float, optional): How many time per seconds do we want to realize the operation. Defaults to 1.
            etl_fn (function, optional): Additional function to transform the data. Defaults to None.
        """

        self.name = name
        self.args = args
        self.tags = tags
        self.interval = interval
        self.etl_fn = etl_fn

        self.influx = InfluxDb(table_name)

    def run(self):
        output = nibi_call(self.args)

        if self.etl_fn is not None:
            output = self.etl_fn(output)

        message = self.influx.Encode(output, time.time_ns(), tags=self.tags)
        status = self.influx.writeEncoded(message)

        if not status["ok"]:
            logger.error(f"Error in operation {self.name}")
            logger.error(status)


class Operations:
    def __init__(self, operations: list[Operation]):
        """
        Create operation

        Args:
            operations (list[Operation]): list of operations to execute
        """

        self.operations = operations

    def launch(self):
        self.scheduler = BackgroundScheduler()
        for operation in self.operations:
            self.scheduler.add_job(
                operation.run, "interval", seconds=operation.interval, max_instances=5
            )

        self.scheduler.start()

