from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="RunWorkItemWorkflowActionRequest")


@_attrs_define
class RunWorkItemWorkflowActionRequest:
    """
    Attributes:
        action_id (str | Unset):
        actor (str | Unset):
        artifact_id (str | Unset):
        reason (str | Unset):
        run_id (str | Unset):
        work_item_id (str | Unset):
    """

    action_id: str | Unset = UNSET
    actor: str | Unset = UNSET
    artifact_id: str | Unset = UNSET
    reason: str | Unset = UNSET
    run_id: str | Unset = UNSET
    work_item_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        action_id = self.action_id

        actor = self.actor

        artifact_id = self.artifact_id

        reason = self.reason

        run_id = self.run_id

        work_item_id = self.work_item_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if action_id is not UNSET:
            field_dict["actionId"] = action_id
        if actor is not UNSET:
            field_dict["actor"] = actor
        if artifact_id is not UNSET:
            field_dict["artifactId"] = artifact_id
        if reason is not UNSET:
            field_dict["reason"] = reason
        if run_id is not UNSET:
            field_dict["runId"] = run_id
        if work_item_id is not UNSET:
            field_dict["workItemId"] = work_item_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action_id = d.pop("actionId", UNSET)

        actor = d.pop("actor", UNSET)

        artifact_id = d.pop("artifactId", UNSET)

        reason = d.pop("reason", UNSET)

        run_id = d.pop("runId", UNSET)

        work_item_id = d.pop("workItemId", UNSET)

        run_work_item_workflow_action_request = cls(
            action_id=action_id,
            actor=actor,
            artifact_id=artifact_id,
            reason=reason,
            run_id=run_id,
            work_item_id=work_item_id,
        )

        run_work_item_workflow_action_request.additional_properties = d
        return run_work_item_workflow_action_request

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
