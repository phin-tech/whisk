from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.start_pty_options_env import StartPTYOptionsEnv


T = TypeVar("T", bound="StartPTYOptions")


@_attrs_define
class StartPTYOptions:
    """
    Attributes:
        cols (int):
        rows (int):
        args (list[str] | Unset):
        command (str | Unset):
        env (StartPTYOptionsEnv | Unset):
        exec_ (bool | Unset):
    """

    cols: int
    rows: int
    args: list[str] | Unset = UNSET
    command: str | Unset = UNSET
    env: StartPTYOptionsEnv | Unset = UNSET
    exec_: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        cols = self.cols

        rows = self.rows

        args: list[str] | Unset = UNSET
        if not isinstance(self.args, Unset):
            args = self.args

        command = self.command

        env: dict[str, Any] | Unset = UNSET
        if not isinstance(self.env, Unset):
            env = self.env.to_dict()

        exec_ = self.exec_

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "cols": cols,
                "rows": rows,
            }
        )
        if args is not UNSET:
            field_dict["args"] = args
        if command is not UNSET:
            field_dict["command"] = command
        if env is not UNSET:
            field_dict["env"] = env
        if exec_ is not UNSET:
            field_dict["exec"] = exec_

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.start_pty_options_env import StartPTYOptionsEnv

        d = dict(src_dict)
        cols = d.pop("cols")

        rows = d.pop("rows")

        args = cast(list[str], d.pop("args", UNSET))

        command = d.pop("command", UNSET)

        _env = d.pop("env", UNSET)
        env: StartPTYOptionsEnv | Unset
        if isinstance(_env, Unset):
            env = UNSET
        else:
            env = StartPTYOptionsEnv.from_dict(_env)

        exec_ = d.pop("exec", UNSET)

        start_pty_options = cls(
            cols=cols,
            rows=rows,
            args=args,
            command=command,
            env=env,
            exec_=exec_,
        )

        start_pty_options.additional_properties = d
        return start_pty_options

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
