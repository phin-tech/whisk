from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.project_workflow import ProjectWorkflow


T = TypeVar("T", bound="Project")


@_attrs_define
class Project:
    """
    Attributes:
        created_at (datetime.datetime):
        id (str):
        name (str):
        next_work_item_number (int):
        root_dir (str):
        slug (str):
        updated_at (datetime.datetime):
        workflow (ProjectWorkflow):
    """

    created_at: datetime.datetime
    id: str
    name: str
    next_work_item_number: int
    root_dir: str
    slug: str
    updated_at: datetime.datetime
    workflow: ProjectWorkflow
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = self.id

        name = self.name

        next_work_item_number = self.next_work_item_number

        root_dir = self.root_dir

        slug = self.slug

        updated_at = self.updated_at.isoformat()

        workflow = self.workflow.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "id": id,
                "name": name,
                "nextWorkItemNumber": next_work_item_number,
                "rootDir": root_dir,
                "slug": slug,
                "updatedAt": updated_at,
                "workflow": workflow,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.project_workflow import ProjectWorkflow

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = d.pop("id")

        name = d.pop("name")

        next_work_item_number = d.pop("nextWorkItemNumber")

        root_dir = d.pop("rootDir")

        slug = d.pop("slug")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        workflow = ProjectWorkflow.from_dict(d.pop("workflow"))

        project = cls(
            created_at=created_at,
            id=id,
            name=name,
            next_work_item_number=next_work_item_number,
            root_dir=root_dir,
            slug=slug,
            updated_at=updated_at,
            workflow=workflow,
        )

        project.additional_properties = d
        return project

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
