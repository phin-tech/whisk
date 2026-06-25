from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="WorkItemLink")


@_attrs_define
class WorkItemLink:
    """
    Attributes:
        created_at (datetime.datetime):
        id (str):
        project_id (str):
        source_work_item_id (str):
        target_work_item_id (str):
        type_ (str):
        created_by (str | Unset):
    """

    created_at: datetime.datetime
    id: str
    project_id: str
    source_work_item_id: str
    target_work_item_id: str
    type_: str
    created_by: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = self.id

        project_id = self.project_id

        source_work_item_id = self.source_work_item_id

        target_work_item_id = self.target_work_item_id

        type_ = self.type_

        created_by = self.created_by

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "id": id,
                "projectId": project_id,
                "sourceWorkItemId": source_work_item_id,
                "targetWorkItemId": target_work_item_id,
                "type": type_,
            }
        )
        if created_by is not UNSET:
            field_dict["createdBy"] = created_by

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = d.pop("id")

        project_id = d.pop("projectId")

        source_work_item_id = d.pop("sourceWorkItemId")

        target_work_item_id = d.pop("targetWorkItemId")

        type_ = d.pop("type")

        created_by = d.pop("createdBy", UNSET)

        work_item_link = cls(
            created_at=created_at,
            id=id,
            project_id=project_id,
            source_work_item_id=source_work_item_id,
            target_work_item_id=target_work_item_id,
            type_=type_,
            created_by=created_by,
        )

        work_item_link.additional_properties = d
        return work_item_link

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
