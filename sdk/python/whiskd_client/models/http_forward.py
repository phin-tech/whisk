from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="HTTPForward")


@_attrs_define
class HTTPForward:
    """
    Attributes:
        id (str):
        name (str):
        session_id (str):
        target_url (str):
    """

    id: str
    name: str
    session_id: str
    target_url: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        session_id = self.session_id

        target_url = self.target_url

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "name": name,
                "sessionId": session_id,
                "targetUrl": target_url,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        session_id = d.pop("sessionId")

        target_url = d.pop("targetUrl")

        http_forward = cls(
            id=id,
            name=name,
            session_id=session_id,
            target_url=target_url,
        )

        http_forward.additional_properties = d
        return http_forward

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
