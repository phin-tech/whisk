from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.work_item import WorkItem


T = TypeVar("T", bound="ReadyWorkItem")


@_attrs_define
class ReadyWorkItem:
    """
    Attributes:
        dependency_count (int):
        dependent_count (int):
        reason (str):
        work_item (WorkItem):
        parent_work_item_id (None | str | Unset):
        resolved_blockers (list[str] | Unset):
    """

    dependency_count: int
    dependent_count: int
    reason: str
    work_item: WorkItem
    parent_work_item_id: None | str | Unset = UNSET
    resolved_blockers: list[str] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        dependency_count = self.dependency_count

        dependent_count = self.dependent_count

        reason = self.reason

        work_item = self.work_item.to_dict()

        parent_work_item_id: None | str | Unset
        if isinstance(self.parent_work_item_id, Unset):
            parent_work_item_id = UNSET
        else:
            parent_work_item_id = self.parent_work_item_id

        resolved_blockers: list[str] | Unset = UNSET
        if not isinstance(self.resolved_blockers, Unset):
            resolved_blockers = self.resolved_blockers

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "dependencyCount": dependency_count,
                "dependentCount": dependent_count,
                "reason": reason,
                "workItem": work_item,
            }
        )
        if parent_work_item_id is not UNSET:
            field_dict["parentWorkItemId"] = parent_work_item_id
        if resolved_blockers is not UNSET:
            field_dict["resolvedBlockers"] = resolved_blockers

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.work_item import WorkItem

        d = dict(src_dict)
        dependency_count = d.pop("dependencyCount")

        dependent_count = d.pop("dependentCount")

        reason = d.pop("reason")

        work_item = WorkItem.from_dict(d.pop("workItem"))

        def _parse_parent_work_item_id(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        parent_work_item_id = _parse_parent_work_item_id(
            d.pop("parentWorkItemId", UNSET)
        )

        resolved_blockers = cast(list[str], d.pop("resolvedBlockers", UNSET))

        ready_work_item = cls(
            dependency_count=dependency_count,
            dependent_count=dependent_count,
            reason=reason,
            work_item=work_item,
            parent_work_item_id=parent_work_item_id,
            resolved_blockers=resolved_blockers,
        )

        ready_work_item.additional_properties = d
        return ready_work_item

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
