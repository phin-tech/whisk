from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.workflow_action_definition import WorkflowActionDefinition


T = TypeVar("T", bound="WorkflowActionAvailability")


@_attrs_define
class WorkflowActionAvailability:
    """
    Attributes:
        action (WorkflowActionDefinition):
        enabled (bool):
        input_kind (str):
        reason (str | Unset):
    """

    action: WorkflowActionDefinition
    enabled: bool
    input_kind: str
    reason: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        action = self.action.to_dict()

        enabled = self.enabled

        input_kind = self.input_kind

        reason = self.reason

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "action": action,
                "enabled": enabled,
                "inputKind": input_kind,
            }
        )
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.workflow_action_definition import WorkflowActionDefinition

        d = dict(src_dict)
        action = WorkflowActionDefinition.from_dict(d.pop("action"))

        enabled = d.pop("enabled")

        input_kind = d.pop("inputKind")

        reason = d.pop("reason", UNSET)

        workflow_action_availability = cls(
            action=action,
            enabled=enabled,
            input_kind=input_kind,
            reason=reason,
        )

        workflow_action_availability.additional_properties = d
        return workflow_action_availability

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
