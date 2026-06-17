from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.update_project_attachment_request_meta import (
        UpdateProjectAttachmentRequestMeta,
    )


T = TypeVar("T", bound="UpdateProjectAttachmentRequest")


@_attrs_define
class UpdateProjectAttachmentRequest:
    """
    Attributes:
        project_id (str):
        include_in_context (bool | None | Unset):
        meta (UpdateProjectAttachmentRequestMeta | Unset):
        note (None | str | Unset):
        path (None | str | Unset):
        provider (None | str | Unset):
        target (None | str | Unset):
        title (None | str | Unset):
        url (None | str | Unset):
    """

    project_id: str
    include_in_context: bool | None | Unset = UNSET
    meta: UpdateProjectAttachmentRequestMeta | Unset = UNSET
    note: None | str | Unset = UNSET
    path: None | str | Unset = UNSET
    provider: None | str | Unset = UNSET
    target: None | str | Unset = UNSET
    title: None | str | Unset = UNSET
    url: None | str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        project_id = self.project_id

        include_in_context: bool | None | Unset
        if isinstance(self.include_in_context, Unset):
            include_in_context = UNSET
        else:
            include_in_context = self.include_in_context

        meta: dict[str, Any] | Unset = UNSET
        if not isinstance(self.meta, Unset):
            meta = self.meta.to_dict()

        note: None | str | Unset
        if isinstance(self.note, Unset):
            note = UNSET
        else:
            note = self.note

        path: None | str | Unset
        if isinstance(self.path, Unset):
            path = UNSET
        else:
            path = self.path

        provider: None | str | Unset
        if isinstance(self.provider, Unset):
            provider = UNSET
        else:
            provider = self.provider

        target: None | str | Unset
        if isinstance(self.target, Unset):
            target = UNSET
        else:
            target = self.target

        title: None | str | Unset
        if isinstance(self.title, Unset):
            title = UNSET
        else:
            title = self.title

        url: None | str | Unset
        if isinstance(self.url, Unset):
            url = UNSET
        else:
            url = self.url

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "projectId": project_id,
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
        from ..models.update_project_attachment_request_meta import (
            UpdateProjectAttachmentRequestMeta,
        )

        d = dict(src_dict)
        project_id = d.pop("projectId")

        def _parse_include_in_context(data: object) -> bool | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(bool | None | Unset, data)

        include_in_context = _parse_include_in_context(d.pop("includeInContext", UNSET))

        _meta = d.pop("meta", UNSET)
        meta: UpdateProjectAttachmentRequestMeta | Unset
        if isinstance(_meta, Unset):
            meta = UNSET
        else:
            meta = UpdateProjectAttachmentRequestMeta.from_dict(_meta)

        def _parse_note(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        note = _parse_note(d.pop("note", UNSET))

        def _parse_path(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        path = _parse_path(d.pop("path", UNSET))

        def _parse_provider(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        provider = _parse_provider(d.pop("provider", UNSET))

        def _parse_target(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        target = _parse_target(d.pop("target", UNSET))

        def _parse_title(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        title = _parse_title(d.pop("title", UNSET))

        def _parse_url(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        url = _parse_url(d.pop("url", UNSET))

        update_project_attachment_request = cls(
            project_id=project_id,
            include_in_context=include_in_context,
            meta=meta,
            note=note,
            path=path,
            provider=provider,
            target=target,
            title=title,
            url=url,
        )

        update_project_attachment_request.additional_properties = d
        return update_project_attachment_request

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
