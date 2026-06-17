from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.attachment_meta import AttachmentMeta


T = TypeVar("T", bound="Attachment")


@_attrs_define
class Attachment:
    """
    Attributes:
        created_at (datetime.datetime):
        id (str):
        kind (str):
        scope (str):
        include_in_context (bool | Unset):
        meta (AttachmentMeta | Unset):
        note (str | Unset):
        path (str | Unset):
        provider (str | Unset):
        target (str | Unset):
        title (str | Unset):
        url (str | Unset):
    """

    created_at: datetime.datetime
    id: str
    kind: str
    scope: str
    include_in_context: bool | Unset = UNSET
    meta: AttachmentMeta | Unset = UNSET
    note: str | Unset = UNSET
    path: str | Unset = UNSET
    provider: str | Unset = UNSET
    target: str | Unset = UNSET
    title: str | Unset = UNSET
    url: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = self.id

        kind = self.kind

        scope = self.scope

        include_in_context = self.include_in_context

        meta: dict[str, Any] | Unset = UNSET
        if not isinstance(self.meta, Unset):
            meta = self.meta.to_dict()

        note = self.note

        path = self.path

        provider = self.provider

        target = self.target

        title = self.title

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
        if include_in_context is not UNSET:
            field_dict["includeInContext"] = include_in_context
        if meta is not UNSET:
            field_dict["meta"] = meta
        if note is not UNSET:
            field_dict["note"] = note
        if path is not UNSET:
            field_dict["path"] = path
        if provider is not UNSET:
            field_dict["provider"] = provider
        if target is not UNSET:
            field_dict["target"] = target
        if title is not UNSET:
            field_dict["title"] = title
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.attachment_meta import AttachmentMeta

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = d.pop("id")

        kind = d.pop("kind")

        scope = d.pop("scope")

        include_in_context = d.pop("includeInContext", UNSET)

        _meta = d.pop("meta", UNSET)
        meta: AttachmentMeta | Unset
        if isinstance(_meta, Unset):
            meta = UNSET
        else:
            meta = AttachmentMeta.from_dict(_meta)

        note = d.pop("note", UNSET)

        path = d.pop("path", UNSET)

        provider = d.pop("provider", UNSET)

        target = d.pop("target", UNSET)

        title = d.pop("title", UNSET)

        url = d.pop("url", UNSET)

        attachment = cls(
            created_at=created_at,
            id=id,
            kind=kind,
            scope=scope,
            include_in_context=include_in_context,
            meta=meta,
            note=note,
            path=path,
            provider=provider,
            target=target,
            title=title,
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
