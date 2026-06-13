from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="GateReport")


@_attrs_define
class GateReport:
    """
    Attributes:
        blocking (bool):
        created_at (datetime.datetime):
        id (str):
        name (str):
        project_id (str):
        status (str):
        updated_at (datetime.datetime):
        work_item_id (str):
        override_reason (str | Unset):
        run_id (str | Unset):
    """

    blocking: bool
    created_at: datetime.datetime
    id: str
    name: str
    project_id: str
    status: str
    updated_at: datetime.datetime
    work_item_id: str
    override_reason: str | Unset = UNSET
    run_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        blocking = self.blocking

        created_at = self.created_at.isoformat()

        id = self.id

        name = self.name

        project_id = self.project_id

        status = self.status

        updated_at = self.updated_at.isoformat()

        work_item_id = self.work_item_id

        override_reason = self.override_reason

        run_id = self.run_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "blocking": blocking,
                "createdAt": created_at,
                "id": id,
                "name": name,
                "projectId": project_id,
                "status": status,
                "updatedAt": updated_at,
                "workItemId": work_item_id,
            }
        )
        if override_reason is not UNSET:
            field_dict["overrideReason"] = override_reason
        if run_id is not UNSET:
            field_dict["runId"] = run_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        blocking = d.pop("blocking")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = d.pop("id")

        name = d.pop("name")

        project_id = d.pop("projectId")

        status = d.pop("status")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        work_item_id = d.pop("workItemId")

        override_reason = d.pop("overrideReason", UNSET)

        run_id = d.pop("runId", UNSET)

        gate_report = cls(
            blocking=blocking,
            created_at=created_at,
            id=id,
            name=name,
            project_id=project_id,
            status=status,
            updated_at=updated_at,
            work_item_id=work_item_id,
            override_reason=override_reason,
            run_id=run_id,
        )

        gate_report.additional_properties = d
        return gate_report

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
