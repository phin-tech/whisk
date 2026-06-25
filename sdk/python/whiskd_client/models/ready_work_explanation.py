from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.blocked_work_item import BlockedWorkItem
    from ..models.ready_work_item import ReadyWorkItem
    from ..models.ready_work_summary import ReadyWorkSummary


T = TypeVar("T", bound="ReadyWorkExplanation")


@_attrs_define
class ReadyWorkExplanation:
    """
    Attributes:
        blocked (list[BlockedWorkItem] | None):
        ready (list[ReadyWorkItem] | None):
        summary (ReadyWorkSummary):
        cycles (list[list[str]] | Unset):
    """

    blocked: list[BlockedWorkItem] | None
    ready: list[ReadyWorkItem] | None
    summary: ReadyWorkSummary
    cycles: list[list[str]] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        blocked: list[dict[str, Any]] | None
        if isinstance(self.blocked, list):
            blocked = []
            for blocked_type_0_item_data in self.blocked:
                blocked_type_0_item = blocked_type_0_item_data.to_dict()
                blocked.append(blocked_type_0_item)

        else:
            blocked = self.blocked

        ready: list[dict[str, Any]] | None
        if isinstance(self.ready, list):
            ready = []
            for ready_type_0_item_data in self.ready:
                ready_type_0_item = ready_type_0_item_data.to_dict()
                ready.append(ready_type_0_item)

        else:
            ready = self.ready

        summary = self.summary.to_dict()

        cycles: list[list[str]] | Unset = UNSET
        if not isinstance(self.cycles, Unset):
            cycles = []
            for cycles_item_data in self.cycles:
                cycles_item = cycles_item_data

                cycles.append(cycles_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "blocked": blocked,
                "ready": ready,
                "summary": summary,
            }
        )
        if cycles is not UNSET:
            field_dict["cycles"] = cycles

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.blocked_work_item import BlockedWorkItem
        from ..models.ready_work_item import ReadyWorkItem
        from ..models.ready_work_summary import ReadyWorkSummary

        d = dict(src_dict)

        def _parse_blocked(data: object) -> list[BlockedWorkItem] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                blocked_type_0 = []
                _blocked_type_0 = data
                for blocked_type_0_item_data in _blocked_type_0:
                    blocked_type_0_item = BlockedWorkItem.from_dict(
                        blocked_type_0_item_data
                    )

                    blocked_type_0.append(blocked_type_0_item)

                return blocked_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[BlockedWorkItem] | None, data)

        blocked = _parse_blocked(d.pop("blocked"))

        def _parse_ready(data: object) -> list[ReadyWorkItem] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                ready_type_0 = []
                _ready_type_0 = data
                for ready_type_0_item_data in _ready_type_0:
                    ready_type_0_item = ReadyWorkItem.from_dict(ready_type_0_item_data)

                    ready_type_0.append(ready_type_0_item)

                return ready_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[ReadyWorkItem] | None, data)

        ready = _parse_ready(d.pop("ready"))

        summary = ReadyWorkSummary.from_dict(d.pop("summary"))

        _cycles = d.pop("cycles", UNSET)
        cycles: list[list[str]] | Unset = UNSET
        if _cycles is not UNSET:
            cycles = []
            for cycles_item_data in _cycles:
                cycles_item = cast(list[str], cycles_item_data)

                cycles.append(cycles_item)

        ready_work_explanation = cls(
            blocked=blocked,
            ready=ready,
            summary=summary,
            cycles=cycles,
        )

        ready_work_explanation.additional_properties = d
        return ready_work_explanation

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
