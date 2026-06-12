from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="ResizePTYRequest")


@_attrs_define
class ResizePTYRequest:
    """
    Attributes:
        cols (int):
        pty_id (str):
        rows (int):
    """

    cols: int
    pty_id: str
    rows: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        cols = self.cols

        pty_id = self.pty_id

        rows = self.rows

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "cols": cols,
                "ptyId": pty_id,
                "rows": rows,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cols = d.pop("cols")

        pty_id = d.pop("ptyId")

        rows = d.pop("rows")

        resize_pty_request = cls(
            cols=cols,
            pty_id=pty_id,
            rows=rows,
        )

        resize_pty_request.additional_properties = d
        return resize_pty_request

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
