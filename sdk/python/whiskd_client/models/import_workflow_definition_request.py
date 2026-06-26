from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.workflow_definition import WorkflowDefinition


T = TypeVar("T", bound="ImportWorkflowDefinitionRequest")


@_attrs_define
class ImportWorkflowDefinitionRequest:
    """
    Attributes:
        definition (WorkflowDefinition):
        source (str | Unset):
        source_path (str | Unset):
    """

    definition: WorkflowDefinition
    source: str | Unset = UNSET
    source_path: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        definition = self.definition.to_dict()

        source = self.source

        source_path = self.source_path

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "definition": definition,
            }
        )
        if source is not UNSET:
            field_dict["source"] = source
        if source_path is not UNSET:
            field_dict["sourcePath"] = source_path

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.workflow_definition import WorkflowDefinition

        d = dict(src_dict)
        definition = WorkflowDefinition.from_dict(d.pop("definition"))

        source = d.pop("source", UNSET)

        source_path = d.pop("sourcePath", UNSET)

        import_workflow_definition_request = cls(
            definition=definition,
            source=source,
            source_path=source_path,
        )

        import_workflow_definition_request.additional_properties = d
        return import_workflow_definition_request

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
