from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ProjectContextItem")


@_attrs_define
class ProjectContextItem:
    """
    Attributes:
        attachment_id (str):
        delivery (str):
        kind (str):
        content (str | Unset):
        content_type (str | Unset):
        error (str | Unset):
        provider (str | Unset):
        source_url (str | Unset):
        target (str | Unset):
        title (str | Unset):
    """

    attachment_id: str
    delivery: str
    kind: str
    content: str | Unset = UNSET
    content_type: str | Unset = UNSET
    error: str | Unset = UNSET
    provider: str | Unset = UNSET
    source_url: str | Unset = UNSET
    target: str | Unset = UNSET
    title: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        attachment_id = self.attachment_id

        delivery = self.delivery

        kind = self.kind

        content = self.content

        content_type = self.content_type

        error = self.error

        provider = self.provider

        source_url = self.source_url

        target = self.target

        title = self.title

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "attachmentId": attachment_id,
                "delivery": delivery,
                "kind": kind,
            }
        )
        if content is not UNSET:
            field_dict["content"] = content
        if content_type is not UNSET:
            field_dict["contentType"] = content_type
        if error is not UNSET:
            field_dict["error"] = error
        if provider is not UNSET:
            field_dict["provider"] = provider
        if source_url is not UNSET:
            field_dict["sourceUrl"] = source_url
        if target is not UNSET:
            field_dict["target"] = target
        if title is not UNSET:
            field_dict["title"] = title

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        attachment_id = d.pop("attachmentId")

        delivery = d.pop("delivery")

        kind = d.pop("kind")

        content = d.pop("content", UNSET)

        content_type = d.pop("contentType", UNSET)

        error = d.pop("error", UNSET)

        provider = d.pop("provider", UNSET)

        source_url = d.pop("sourceUrl", UNSET)

        target = d.pop("target", UNSET)

        title = d.pop("title", UNSET)

        project_context_item = cls(
            attachment_id=attachment_id,
            delivery=delivery,
            kind=kind,
            content=content,
            content_type=content_type,
            error=error,
            provider=provider,
            source_url=source_url,
            target=target,
            title=title,
        )

        project_context_item.additional_properties = d
        return project_context_item

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
