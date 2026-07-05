from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="BrowserTarget")


@_attrs_define
class BrowserTarget:
    """
    Attributes:
        id (str):
        resource_id (str):
        status (str):
        type_ (str):
        title (str | Unset):
        url (str | Unset):
    """

    id: str
    resource_id: str
    status: str
    type_: str
    title: str | Unset = UNSET
    url: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        resource_id = self.resource_id

        status = self.status

        type_ = self.type_

        title = self.title

        url = self.url

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "resourceId": resource_id,
                "status": status,
                "type": type_,
            }
        )
        if title is not UNSET:
            field_dict["title"] = title
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        resource_id = d.pop("resourceId")

        status = d.pop("status")

        type_ = d.pop("type")

        title = d.pop("title", UNSET)

        url = d.pop("url", UNSET)

        browser_target = cls(
            id=id,
            resource_id=resource_id,
            status=status,
            type_=type_,
            title=title,
            url=url,
        )

        browser_target.additional_properties = d
        return browser_target

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
