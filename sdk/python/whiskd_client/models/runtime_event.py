from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="RuntimeEvent")


@_attrs_define
class RuntimeEvent:
    """
    Attributes:
        type_ (str):
        offset (int | Unset):
        pty_id (str | Unset):
    """

    type_: str
    offset: int | Unset = UNSET
    pty_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        type_ = self.type_

        offset = self.offset

        pty_id = self.pty_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "type": type_,
            }
        )
        if offset is not UNSET:
            field_dict["offset"] = offset
        if pty_id is not UNSET:
            field_dict["ptyId"] = pty_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        type_ = d.pop("type")

        offset = d.pop("offset", UNSET)

        pty_id = d.pop("ptyId", UNSET)

        runtime_event = cls(
            type_=type_,
            offset=offset,
            pty_id=pty_id,
        )

        runtime_event.additional_properties = d
        return runtime_event

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
