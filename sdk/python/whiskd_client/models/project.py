from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.attachment import Attachment
    from ..models.project_metadata import ProjectMetadata
    from ..models.project_preferences import ProjectPreferences
    from ..models.project_workflow import ProjectWorkflow


T = TypeVar("T", bound="Project")


@_attrs_define
class Project:
    """
    Attributes:
        attachments (list[Attachment] | None):
        created_at (datetime.datetime):
        id (str):
        name (str):
        next_work_item_number (int):
        preferences (ProjectPreferences):
        root_dir (str):
        slug (str):
        updated_at (datetime.datetime):
        workflow (ProjectWorkflow):
        description (str | Unset):
        metadata (ProjectMetadata | Unset):
    """

    attachments: list[Attachment] | None
    created_at: datetime.datetime
    id: str
    name: str
    next_work_item_number: int
    preferences: ProjectPreferences
    root_dir: str
    slug: str
    updated_at: datetime.datetime
    workflow: ProjectWorkflow
    description: str | Unset = UNSET
    metadata: ProjectMetadata | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        attachments: list[dict[str, Any]] | None
        if isinstance(self.attachments, list):
            attachments = []
            for attachments_type_0_item_data in self.attachments:
                attachments_type_0_item = attachments_type_0_item_data.to_dict()
                attachments.append(attachments_type_0_item)

        else:
            attachments = self.attachments

        created_at = self.created_at.isoformat()

        id = self.id

        name = self.name

        next_work_item_number = self.next_work_item_number

        preferences = self.preferences.to_dict()

        root_dir = self.root_dir

        slug = self.slug

        updated_at = self.updated_at.isoformat()

        workflow = self.workflow.to_dict()

        description = self.description

        metadata: dict[str, Any] | Unset = UNSET
        if not isinstance(self.metadata, Unset):
            metadata = self.metadata.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "attachments": attachments,
                "createdAt": created_at,
                "id": id,
                "name": name,
                "nextWorkItemNumber": next_work_item_number,
                "preferences": preferences,
                "rootDir": root_dir,
                "slug": slug,
                "updatedAt": updated_at,
                "workflow": workflow,
            }
        )
        if description is not UNSET:
            field_dict["description"] = description
        if metadata is not UNSET:
            field_dict["metadata"] = metadata

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.attachment import Attachment
        from ..models.project_metadata import ProjectMetadata
        from ..models.project_preferences import ProjectPreferences
        from ..models.project_workflow import ProjectWorkflow

        d = dict(src_dict)

        def _parse_attachments(data: object) -> list[Attachment] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                attachments_type_0 = []
                _attachments_type_0 = data
                for attachments_type_0_item_data in _attachments_type_0:
                    attachments_type_0_item = Attachment.from_dict(
                        attachments_type_0_item_data
                    )

                    attachments_type_0.append(attachments_type_0_item)

                return attachments_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[Attachment] | None, data)

        attachments = _parse_attachments(d.pop("attachments"))

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = d.pop("id")

        name = d.pop("name")

        next_work_item_number = d.pop("nextWorkItemNumber")

        preferences = ProjectPreferences.from_dict(d.pop("preferences"))

        root_dir = d.pop("rootDir")

        slug = d.pop("slug")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        workflow = ProjectWorkflow.from_dict(d.pop("workflow"))

        description = d.pop("description", UNSET)

        _metadata = d.pop("metadata", UNSET)
        metadata: ProjectMetadata | Unset
        if isinstance(_metadata, Unset):
            metadata = UNSET
        else:
            metadata = ProjectMetadata.from_dict(_metadata)

        project = cls(
            attachments=attachments,
            created_at=created_at,
            id=id,
            name=name,
            next_work_item_number=next_work_item_number,
            preferences=preferences,
            root_dir=root_dir,
            slug=slug,
            updated_at=updated_at,
            workflow=workflow,
            description=description,
            metadata=metadata,
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
