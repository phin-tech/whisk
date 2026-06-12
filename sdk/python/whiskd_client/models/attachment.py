from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Attachment")


@_attrs_define
class Attachment:
    """
    Attributes:
        created_at (datetime.datetime):
        id (str):
        kind (str):
        scope (str):
        note (str | Unset):
        path (str | Unset):
        url (str | Unset):
    """

    created_at: datetime.datetime
    id: str
    kind: str
    scope: str
    note: str | Unset = UNSET
    path: str | Unset = UNSET
    url: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = self.id

        kind = self.kind

        scope = self.scope

        note = self.note

        path = self.path

        url = self.url

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "id": id,
                "kind": kind,
                "scope": scope,
            }
        )
        if note is not UNSET:
            field_dict["note"] = note
        if path is not UNSET:
            field_dict["path"] = path
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = d.pop("id")

        kind = d.pop("kind")

        scope = d.pop("scope")

        note = d.pop("note", UNSET)

        path = d.pop("path", UNSET)

        url = d.pop("url", UNSET)

        attachment = cls(
            created_at=created_at,
            id=id,
            kind=kind,
            scope=scope,
            note=note,
            path=path,
            url=url,
        )

        attachment.additional_properties = d
        return attachment

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
