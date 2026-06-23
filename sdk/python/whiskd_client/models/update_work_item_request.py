from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="UpdateWorkItemRequest")


@_attrs_define
class UpdateWorkItemRequest:
    """
    Attributes:
        id (str):
        actor (str | Unset):
        body_markdown (None | str | Unset):
        title (None | str | Unset):
    """

    id: str
    actor: str | Unset = UNSET
    body_markdown: None | str | Unset = UNSET
    title: None | str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        actor = self.actor

        body_markdown: None | str | Unset
        if isinstance(self.body_markdown, Unset):
            body_markdown = UNSET
        else:
            body_markdown = self.body_markdown

        title: None | str | Unset
        if isinstance(self.title, Unset):
            title = UNSET
        else:
            title = self.title

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if body_markdown is not UNSET:
            field_dict["bodyMarkdown"] = body_markdown
        if title is not UNSET:
            field_dict["title"] = title

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        actor = d.pop("actor", UNSET)

        def _parse_body_markdown(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        body_markdown = _parse_body_markdown(d.pop("bodyMarkdown", UNSET))

        def _parse_title(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        title = _parse_title(d.pop("title", UNSET))

        update_work_item_request = cls(
            id=id,
            actor=actor,
            body_markdown=body_markdown,
            title=title,
        )

        update_work_item_request.additional_properties = d
        return update_work_item_request

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
