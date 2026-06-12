from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="AddWorkItemAttachmentRequest")


@_attrs_define
class AddWorkItemAttachmentRequest:
    """
    Attributes:
        kind (str):
        work_item_id (str):
        actor (str | Unset):
        note (str | Unset):
        path (str | Unset):
        scope (str | Unset):
        url (str | Unset):
    """

    kind: str
    work_item_id: str
    actor: str | Unset = UNSET
    note: str | Unset = UNSET
    path: str | Unset = UNSET
    scope: str | Unset = UNSET
    url: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        kind = self.kind

        work_item_id = self.work_item_id

        actor = self.actor

        note = self.note

        path = self.path

        scope = self.scope

        url = self.url

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "kind": kind,
                "workItemId": work_item_id,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if note is not UNSET:
            field_dict["note"] = note
        if path is not UNSET:
            field_dict["path"] = path
        if scope is not UNSET:
            field_dict["scope"] = scope
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = d.pop("kind")

        work_item_id = d.pop("workItemId")

        actor = d.pop("actor", UNSET)

        note = d.pop("note", UNSET)

        path = d.pop("path", UNSET)

        scope = d.pop("scope", UNSET)

        url = d.pop("url", UNSET)

        add_work_item_attachment_request = cls(
            kind=kind,
            work_item_id=work_item_id,
            actor=actor,
            note=note,
            path=path,
            scope=scope,
            url=url,
        )

        add_work_item_attachment_request.additional_properties = d
        return add_work_item_attachment_request

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
