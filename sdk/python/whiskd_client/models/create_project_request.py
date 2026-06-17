from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.project_preferences import ProjectPreferences


T = TypeVar("T", bound="CreateProjectRequest")


@_attrs_define
class CreateProjectRequest:
    """
    Attributes:
        name (str):
        root_dir (str):
        description (str | Unset):
        preferences (ProjectPreferences | Unset):
        slug (str | Unset):
        workflow_id (str | Unset):
    """

    name: str
    root_dir: str
    description: str | Unset = UNSET
    preferences: ProjectPreferences | Unset = UNSET
    slug: str | Unset = UNSET
    workflow_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        root_dir = self.root_dir

        description = self.description

        preferences: dict[str, Any] | Unset = UNSET
        if not isinstance(self.preferences, Unset):
            preferences = self.preferences.to_dict()

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
        if description is not UNSET:
            field_dict["description"] = description
        if preferences is not UNSET:
            field_dict["preferences"] = preferences
        if slug is not UNSET:
            field_dict["slug"] = slug
        if workflow_id is not UNSET:
            field_dict["workflowId"] = workflow_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.project_preferences import ProjectPreferences

        d = dict(src_dict)
        name = d.pop("name")

        root_dir = d.pop("rootDir")

        description = d.pop("description", UNSET)

        _preferences = d.pop("preferences", UNSET)
        preferences: ProjectPreferences | Unset
        if isinstance(_preferences, Unset):
            preferences = UNSET
        else:
            preferences = ProjectPreferences.from_dict(_preferences)

        slug = d.pop("slug", UNSET)

        workflow_id = d.pop("workflowId", UNSET)

        create_project_request = cls(
            name=name,
            root_dir=root_dir,
            description=description,
            preferences=preferences,
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
