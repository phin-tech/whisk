from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="CreateWorkItemRequest")


@_attrs_define
class CreateWorkItemRequest:
    """
    Attributes:
        project_id (str):
        title (str):
        actor (str | Unset):
        body_markdown (str | Unset):
        stage_id (str | Unset):
    """

    project_id: str
    title: str
    actor: str | Unset = UNSET
    body_markdown: str | Unset = UNSET
    stage_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        project_id = self.project_id

        title = self.title

        actor = self.actor

        body_markdown = self.body_markdown

        stage_id = self.stage_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "projectId": project_id,
                "title": title,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if body_markdown is not UNSET:
            field_dict["bodyMarkdown"] = body_markdown
        if stage_id is not UNSET:
            field_dict["stageId"] = stage_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        project_id = d.pop("projectId")

        title = d.pop("title")

        actor = d.pop("actor", UNSET)

        body_markdown = d.pop("bodyMarkdown", UNSET)

        stage_id = d.pop("stageId", UNSET)

        create_work_item_request = cls(
            project_id=project_id,
            title=title,
            actor=actor,
            body_markdown=body_markdown,
            stage_id=stage_id,
        )

        create_work_item_request.additional_properties = d
        return create_work_item_request

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
