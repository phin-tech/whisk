from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.ready_blocker_info import ReadyBlockerInfo
    from ..models.work_item import WorkItem


T = TypeVar("T", bound="BlockedWorkItem")


@_attrs_define
class BlockedWorkItem:
    """
    Attributes:
        blocked_by (list[ReadyBlockerInfo] | None):
        blocked_by_count (int):
        work_item (WorkItem):
    """

    blocked_by: list[ReadyBlockerInfo] | None
    blocked_by_count: int
    work_item: WorkItem
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        blocked_by: list[dict[str, Any]] | None
        if isinstance(self.blocked_by, list):
            blocked_by = []
            for blocked_by_type_0_item_data in self.blocked_by:
                blocked_by_type_0_item = blocked_by_type_0_item_data.to_dict()
                blocked_by.append(blocked_by_type_0_item)

        else:
            blocked_by = self.blocked_by

        blocked_by_count = self.blocked_by_count

        work_item = self.work_item.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "blockedBy": blocked_by,
                "blockedByCount": blocked_by_count,
                "workItem": work_item,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ready_blocker_info import ReadyBlockerInfo
        from ..models.work_item import WorkItem

        d = dict(src_dict)

        def _parse_blocked_by(data: object) -> list[ReadyBlockerInfo] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                blocked_by_type_0 = []
                _blocked_by_type_0 = data
                for blocked_by_type_0_item_data in _blocked_by_type_0:
                    blocked_by_type_0_item = ReadyBlockerInfo.from_dict(
                        blocked_by_type_0_item_data
                    )

                    blocked_by_type_0.append(blocked_by_type_0_item)

                return blocked_by_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[ReadyBlockerInfo] | None, data)

        blocked_by = _parse_blocked_by(d.pop("blockedBy"))

        blocked_by_count = d.pop("blockedByCount")

        work_item = WorkItem.from_dict(d.pop("workItem"))

        blocked_work_item = cls(
            blocked_by=blocked_by,
            blocked_by_count=blocked_by_count,
            work_item=work_item,
        )

        blocked_work_item.additional_properties = d
        return blocked_work_item

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
