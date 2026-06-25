from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="ReadyWorkSummary")


@_attrs_define
class ReadyWorkSummary:
    """
    Attributes:
        cycle_count (int):
        total_blocked (int):
        total_ready (int):
    """

    cycle_count: int
    total_blocked: int
    total_ready: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        cycle_count = self.cycle_count

        total_blocked = self.total_blocked

        total_ready = self.total_ready

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "cycleCount": cycle_count,
                "totalBlocked": total_blocked,
                "totalReady": total_ready,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cycle_count = d.pop("cycleCount")

        total_blocked = d.pop("totalBlocked")

        total_ready = d.pop("totalReady")

        ready_work_summary = cls(
            cycle_count=cycle_count,
            total_blocked=total_blocked,
            total_ready=total_ready,
        )

        ready_work_summary.additional_properties = d
        return ready_work_summary

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
