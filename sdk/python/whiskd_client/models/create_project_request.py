from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="CreateProjectRequest")


@_attrs_define
class CreateProjectRequest:
    """
    Attributes:
        name (str):
        root_dir (str):
        slug (str | Unset):
        workflow_id (str | Unset):
    """

    name: str
    root_dir: str
    slug: str | Unset = UNSET
    workflow_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        root_dir = self.root_dir

        slug = self.slug

        workflow_id = self.workflow_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "rootDir": root_dir,
            }
        )
        if slug is not UNSET:
            field_dict["slug"] = slug
        if workflow_id is not UNSET:
            field_dict["workflowId"] = workflow_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        root_dir = d.pop("rootDir")

        slug = d.pop("slug", UNSET)

        workflow_id = d.pop("workflowId", UNSET)

        create_project_request = cls(
            name=name,
            root_dir=root_dir,
            slug=slug,
            workflow_id=workflow_id,
        )

        create_project_request.additional_properties = d
        return create_project_request

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
