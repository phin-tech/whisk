from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="AddPTYBookmarkRequest")


@_attrs_define
class AddPTYBookmarkRequest:
    """
    Attributes:
        kind (str):
        label (str):
        offset (int):
        pty_id (str):
    """

    kind: str
    label: str
    offset: int
    pty_id: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        kind = self.kind

        label = self.label

        offset = self.offset

        pty_id = self.pty_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "kind": kind,
                "label": label,
                "offset": offset,
                "ptyId": pty_id,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = d.pop("kind")

        label = d.pop("label")

        offset = d.pop("offset")

        pty_id = d.pop("ptyId")

        add_pty_bookmark_request = cls(
            kind=kind,
            label=label,
            offset=offset,
            pty_id=pty_id,
        )

        add_pty_bookmark_request.additional_properties = d
        return add_pty_bookmark_request

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
