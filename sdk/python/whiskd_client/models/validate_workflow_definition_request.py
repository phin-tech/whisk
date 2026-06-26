from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.workflow_definition import WorkflowDefinition


T = TypeVar("T", bound="ValidateWorkflowDefinitionRequest")


@_attrs_define
class ValidateWorkflowDefinitionRequest:
    """
    Attributes:
        definition (WorkflowDefinition):
    """

    definition: WorkflowDefinition
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        definition = self.definition.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "definition": definition,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.workflow_definition import WorkflowDefinition

        d = dict(src_dict)
        definition = WorkflowDefinition.from_dict(d.pop("definition"))

        validate_workflow_definition_request = cls(
            definition=definition,
        )

        validate_workflow_definition_request.additional_properties = d
        return validate_workflow_definition_request

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
