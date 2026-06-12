from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.artifact_metadata import ArtifactMetadata


T = TypeVar("T", bound="Artifact")


@_attrs_define
class Artifact:
    """
    Attributes:
        created_at (datetime.datetime):
        id (str):
        kind (str):
        project_id (str):
        status (str):
        updated_at (datetime.datetime):
        work_item_id (str):
        body (str | Unset):
        metadata (ArtifactMetadata | Unset):
        run_id (str | Unset):
        title (str | Unset):
    """

    created_at: datetime.datetime
    id: str
    kind: str
    project_id: str
    status: str
    updated_at: datetime.datetime
    work_item_id: str
    body: str | Unset = UNSET
    metadata: ArtifactMetadata | Unset = UNSET
    run_id: str | Unset = UNSET
    title: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = self.id

        kind = self.kind

        project_id = self.project_id

        status = self.status

        updated_at = self.updated_at.isoformat()

        work_item_id = self.work_item_id

        body = self.body

        metadata: dict[str, Any] | Unset = UNSET
        if not isinstance(self.metadata, Unset):
            metadata = self.metadata.to_dict()

        run_id = self.run_id

        title = self.title

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "id": id,
                "kind": kind,
                "projectId": project_id,
                "status": status,
                "updatedAt": updated_at,
                "workItemId": work_item_id,
            }
        )
        if body is not UNSET:
            field_dict["body"] = body
        if metadata is not UNSET:
            field_dict["metadata"] = metadata
        if run_id is not UNSET:
            field_dict["runId"] = run_id
        if title is not UNSET:
            field_dict["title"] = title

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.artifact_metadata import ArtifactMetadata

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = d.pop("id")

        kind = d.pop("kind")

        project_id = d.pop("projectId")

        status = d.pop("status")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        work_item_id = d.pop("workItemId")

        body = d.pop("body", UNSET)

        _metadata = d.pop("metadata", UNSET)
        metadata: ArtifactMetadata | Unset
        if isinstance(_metadata, Unset):
            metadata = UNSET
        else:
            metadata = ArtifactMetadata.from_dict(_metadata)

        run_id = d.pop("runId", UNSET)

        title = d.pop("title", UNSET)

        artifact = cls(
            created_at=created_at,
            id=id,
            kind=kind,
            project_id=project_id,
            status=status,
            updated_at=updated_at,
            work_item_id=work_item_id,
            body=body,
            metadata=metadata,
            run_id=run_id,
            title=title,
        )

        artifact.additional_properties = d
        return artifact

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
