from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="TransitionRule")


@_attrs_define
class TransitionRule:
    """
    Attributes:
        from_stage_id (str):
        to_stage_id (str):
        requires_approval (bool | Unset):
        requires_checks (bool | Unset):
        requires_no_running_runs (bool | Unset):
    """

    from_stage_id: str
    to_stage_id: str
    requires_approval: bool | Unset = UNSET
    requires_checks: bool | Unset = UNSET
    requires_no_running_runs: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from_stage_id = self.from_stage_id

        to_stage_id = self.to_stage_id

        requires_approval = self.requires_approval

        requires_checks = self.requires_checks

        requires_no_running_runs = self.requires_no_running_runs

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "fromStageId": from_stage_id,
                "toStageId": to_stage_id,
            }
        )
        if requires_approval is not UNSET:
            field_dict["requiresApproval"] = requires_approval
        if requires_checks is not UNSET:
            field_dict["requiresChecks"] = requires_checks
        if requires_no_running_runs is not UNSET:
            field_dict["requiresNoRunningRuns"] = requires_no_running_runs

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        from_stage_id = d.pop("fromStageId")

        to_stage_id = d.pop("toStageId")

        requires_approval = d.pop("requiresApproval", UNSET)

        requires_checks = d.pop("requiresChecks", UNSET)

        requires_no_running_runs = d.pop("requiresNoRunningRuns", UNSET)

        transition_rule = cls(
            from_stage_id=from_stage_id,
            to_stage_id=to_stage_id,
            requires_approval=requires_approval,
            requires_checks=requires_checks,
            requires_no_running_runs=requires_no_running_runs,
        )

        transition_rule.additional_properties = d
        return transition_rule

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
