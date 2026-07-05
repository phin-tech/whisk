from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="BrowserResource")


@_attrs_define
class BrowserResource:
    """
    Attributes:
        cdp_url (str):
        connected (bool):
        id (str):
        name (str | Unset):
    """

    cdp_url: str
    connected: bool
    id: str
    name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        cdp_url = self.cdp_url

        connected = self.connected

        id = self.id

        name = self.name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "cdpUrl": cdp_url,
                "connected": connected,
                "id": id,
            }
        )
        if name is not UNSET:
            field_dict["name"] = name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cdp_url = d.pop("cdpUrl")

        connected = d.pop("connected")

        id = d.pop("id")

        name = d.pop("name", UNSET)

        browser_resource = cls(
            cdp_url=cdp_url,
            connected=connected,
            id=id,
            name=name,
        )

        browser_resource.additional_properties = d
        return browser_resource

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
