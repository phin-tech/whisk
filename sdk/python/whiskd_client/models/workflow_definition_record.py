from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.workflow_definition import WorkflowDefinition


T = TypeVar("T", bound="WorkflowDefinitionRecord")


@_attrs_define
class WorkflowDefinitionRecord:
    """
    Attributes:
        content_hash (str):
        created_at (datetime.datetime):
        definition (WorkflowDefinition):
        id (str):
        source (str):
        updated_at (datetime.datetime):
        version (int):
        source_path (str | Unset):
    """

    content_hash: str
    created_at: datetime.datetime
    definition: WorkflowDefinition
    id: str
    source: str
    updated_at: datetime.datetime
    version: int
    source_path: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        content_hash = self.content_hash

        created_at = self.created_at.isoformat()

        definition = self.definition.to_dict()

        id = self.id

        source = self.source

        updated_at = self.updated_at.isoformat()

        version = self.version

        source_path = self.source_path

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "contentHash": content_hash,
                "createdAt": created_at,
                "definition": definition,
                "id": id,
                "source": source,
                "updatedAt": updated_at,
                "version": version,
            }
        )
        if source_path is not UNSET:
            field_dict["sourcePath"] = source_path

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.workflow_definition import WorkflowDefinition

        d = dict(src_dict)
        content_hash = d.pop("contentHash")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        definition = WorkflowDefinition.from_dict(d.pop("definition"))

        id = d.pop("id")

        source = d.pop("source")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        version = d.pop("version")

        source_path = d.pop("sourcePath", UNSET)

        workflow_definition_record = cls(
            content_hash=content_hash,
            created_at=created_at,
            definition=definition,
            id=id,
            source=source,
            updated_at=updated_at,
            version=version,
            source_path=source_path,
        )

        workflow_definition_record.additional_properties = d
        return workflow_definition_record

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
