from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="WorkflowGateDefinition")


@_attrs_define
class WorkflowGateDefinition:
    """
    Attributes:
        blocking (bool):
        id (str):
        phase (str):
    """

    blocking: bool
    id: str
    phase: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        blocking = self.blocking

        id = self.id

        phase = self.phase

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "blocking": blocking,
                "id": id,
                "phase": phase,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        blocking = d.pop("blocking")

        id = d.pop("id")

        phase = d.pop("phase")

        workflow_gate_definition = cls(
            blocking=blocking,
            id=id,
            phase=phase,
        )

        workflow_gate_definition.additional_properties = d
        return workflow_gate_definition

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
