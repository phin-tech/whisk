from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="StartPTYOptions")


@_attrs_define
class StartPTYOptions:
    """
    Attributes:
        cols (int):
        rows (int):
        command (str | Unset):
    """

    cols: int
    rows: int
    command: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        cols = self.cols

        rows = self.rows

        command = self.command

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "cols": cols,
                "rows": rows,
            }
        )
        if command is not UNSET:
            field_dict["command"] = command

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cols = d.pop("cols")

        rows = d.pop("rows")

        command = d.pop("command", UNSET)

        start_pty_options = cls(
            cols=cols,
            rows=rows,
            command=command,
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
