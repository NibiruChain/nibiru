import json
import traceback
from decimal import Decimal
from typing import Mapping, Optional
from datetime import datetime

from influxdb_client import InfluxDBClient
from lib.constants import KEY, influx_credential
from influxdb_client.client.write_api import SYNCHRONOUS

DEFAULT_TABLE = "DEFAULT_SERVICE"
DEFAULT_DATABASE = "DEFAULT_DATABASE"
DEFAULT_SYMBOL = "DEFAULT_SYMBOL"
DEFAULT_EXCHANGE = "DEFAULT_EXCHANGE"

FORMAT = "%Y-%m-%dT%H:%M:%S.%f%z"


def load_list(l: list) -> list:
    result = []
    for item in l:
        if isinstance(item, str):
            try:
                result.append(Decimal(item))
            except:
                result.append(item)
        elif isinstance(item, list):
            result.append(load_list(item))
    return result


def custom_load(o):
    for key, value in o.items():
        try:
            o[key] = datetime.strptime(value, FORMAT)
        except:
            if isinstance(value, str):
                try:
                    o[key] = Decimal(value)
                except:
                    o[key] = value
            elif isinstance(value, list):
                o[key] = load_list(value)
    return o


class InfluxDb:
    def __init__(self, table_name):
        self._table = table_name
        self._database = influx_credential.get(KEY.DATABASE, DEFAULT_DATABASE)
        self._header = f"{self._table}"

        # Open database connection
        self._client = InfluxDBClient(**influx_credential)

    # NOTE: Code is pretty UNSAFE in order to speed-uo
    def Encode(self, fields: Mapping[str, any], timestamp: int, tags: list = []):

        fields = fields.copy()

        def fn(value):
            if isinstance(value, bool):
                return value
            elif isinstance(value, int):
                return f"{value}"
            elif isinstance(value, Decimal):
                return float(value)
            elif isinstance(value, float):
                return value
            else:
                return f'"{str(value)}"'

        header = f"{self._table}"

        for tag in tags:
            header += f",{tag}={fields[tag]}"
            fields.pop(tag)

        body = ",".join(
            [f"{key}={fn(value)}" for key, value in fields.items() if value is not None]
        )

        return f"{header} {body} {int(timestamp)}"

    def writeEncoded(self, data: list):
        try:
            self._client.write_api(write_options=SYNCHRONOUS).write(
                "report", "admin", data, params=dict(db=self._database), protocol="line"
            )
            return {
                "ok": True,
            }
        except Exception as e:
            return {
                "ok": False,
                "exception": e,
                "traceback": traceback.format_exc(),
            }

    def readLast(self, field: str):
        query = (
            f'select {field} from "{self._table}" '
            f"where \"symbol\"='{self._symbol}' AND \"exchange\"='{self._exchange}' "
            f"order by time desc limit 1"
        )

        reply = self._client.query(query)

        for item in reply.get_points():
            data = item[field]
            try:
                return json.loads(data, object_hook=custom_load)
            except:
                return data
