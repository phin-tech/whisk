from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.status_event import StatusEvent
    from ..models.work_item import WorkItem
    from ..models.work_item_run import WorkItemRun


T = TypeVar("T", bound="ReportStatusResponse")


@_attrs_define
class ReportStatusResponse:
    """
    Attributes:
        event (StatusEvent):
        run (None | Unset | WorkItemRun):
        work_item (None | Unset | WorkItem):
    """

    event: StatusEvent
    run: None | Unset | WorkItemRun = UNSET
    work_item: None | Unset | WorkItem = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.work_item import WorkItem
        from ..models.work_item_run import WorkItemRun

        event = self.event.to_dict()

        run: dict[str, Any] | None | Unset
        if isinstance(self.run, Unset):
            run = UNSET
        elif isinstance(self.run, WorkItemRun):
            run = self.run.to_dict()
        else:
            run = self.run

        work_item: dict[str, Any] | None | Unset
        if isinstance(self.work_item, Unset):
            work_item = UNSET
        elif isinstance(self.work_item, WorkItem):
            work_item = self.work_item.to_dict()
        else:
            work_item = self.work_item

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "event": event,
            }
        )
        if run is not UNSET:
            field_dict["run"] = run
        if work_item is not UNSET:
            field_dict["workItem"] = work_item

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.status_event import StatusEvent
        from ..models.work_item import WorkItem
        from ..models.work_item_run import WorkItemRun

        d = dict(src_dict)
        event = StatusEvent.from_dict(d.pop("event"))

        def _parse_run(data: object) -> None | Unset | WorkItemRun:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                run_type_1 = WorkItemRun.from_dict(data)

                return run_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | WorkItemRun, data)

        run = _parse_run(d.pop("run", UNSET))

        def _parse_work_item(data: object) -> None | Unset | WorkItem:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                work_item_type_1 = WorkItem.from_dict(data)

                return work_item_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | WorkItem, data)

        work_item = _parse_work_item(d.pop("workItem", UNSET))

        report_status_response = cls(
            event=event,
            run=run,
            work_item=work_item,
        )

        report_status_response.additional_properties = d
        return report_status_response

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
