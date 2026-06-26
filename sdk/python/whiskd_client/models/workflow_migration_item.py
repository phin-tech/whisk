from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="WorkflowMigrationItem")


@_attrs_define
class WorkflowMigrationItem:
    """
    Attributes:
        compatible (bool):
        current_stage_id (str):
        current_workflow_id (str):
        current_workflow_version (int):
        number (int):
        title (str):
        work_item_id (str):
        reason (str | Unset):
        target_stage_id (str | Unset):
    """

    compatible: bool
    current_stage_id: str
    current_workflow_id: str
    current_workflow_version: int
    number: int
    title: str
    work_item_id: str
    reason: str | Unset = UNSET
    target_stage_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        compatible = self.compatible

        current_stage_id = self.current_stage_id

        current_workflow_id = self.current_workflow_id

        current_workflow_version = self.current_workflow_version

        number = self.number

        title = self.title

        work_item_id = self.work_item_id

        reason = self.reason

        target_stage_id = self.target_stage_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "compatible": compatible,
                "currentStageId": current_stage_id,
                "currentWorkflowId": current_workflow_id,
                "currentWorkflowVersion": current_workflow_version,
                "number": number,
                "title": title,
                "workItemId": work_item_id,
            }
        )
        if reason is not UNSET:
            field_dict["reason"] = reason
        if target_stage_id is not UNSET:
            field_dict["targetStageId"] = target_stage_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        compatible = d.pop("compatible")

        current_stage_id = d.pop("currentStageId")

        current_workflow_id = d.pop("currentWorkflowId")

        current_workflow_version = d.pop("currentWorkflowVersion")

        number = d.pop("number")

        title = d.pop("title")

        work_item_id = d.pop("workItemId")

        reason = d.pop("reason", UNSET)

        target_stage_id = d.pop("targetStageId", UNSET)

        workflow_migration_item = cls(
            compatible=compatible,
            current_stage_id=current_stage_id,
            current_workflow_id=current_workflow_id,
            current_workflow_version=current_workflow_version,
            number=number,
            title=title,
            work_item_id=work_item_id,
            reason=reason,
            target_stage_id=target_stage_id,
        )

        workflow_migration_item.additional_properties = d
        return workflow_migration_item

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
