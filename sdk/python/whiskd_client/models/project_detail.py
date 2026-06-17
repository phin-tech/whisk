from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.project import Project
    from ..models.session import Session
    from ..models.work_item import WorkItem
    from ..models.work_item_run import WorkItemRun


T = TypeVar("T", bound="ProjectDetail")


@_attrs_define
class ProjectDetail:
    """
    Attributes:
        project (Project):
        runs (list[WorkItemRun] | None):
        sessions (list[Session] | None):
        work_items (list[WorkItem] | None):
    """

    project: Project
    runs: list[WorkItemRun] | None
    sessions: list[Session] | None
    work_items: list[WorkItem] | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        project = self.project.to_dict()

        runs: list[dict[str, Any]] | None
        if isinstance(self.runs, list):
            runs = []
            for runs_type_0_item_data in self.runs:
                runs_type_0_item = runs_type_0_item_data.to_dict()
                runs.append(runs_type_0_item)

        else:
            runs = self.runs

        sessions: list[dict[str, Any]] | None
        if isinstance(self.sessions, list):
            sessions = []
            for sessions_type_0_item_data in self.sessions:
                sessions_type_0_item = sessions_type_0_item_data.to_dict()
                sessions.append(sessions_type_0_item)

        else:
            sessions = self.sessions

        work_items: list[dict[str, Any]] | None
        if isinstance(self.work_items, list):
            work_items = []
            for work_items_type_0_item_data in self.work_items:
                work_items_type_0_item = work_items_type_0_item_data.to_dict()
                work_items.append(work_items_type_0_item)

        else:
            work_items = self.work_items

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "project": project,
                "runs": runs,
                "sessions": sessions,
                "workItems": work_items,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.project import Project
        from ..models.session import Session
        from ..models.work_item import WorkItem
        from ..models.work_item_run import WorkItemRun

        d = dict(src_dict)
        project = Project.from_dict(d.pop("project"))

        def _parse_runs(data: object) -> list[WorkItemRun] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                runs_type_0 = []
                _runs_type_0 = data
                for runs_type_0_item_data in _runs_type_0:
                    runs_type_0_item = WorkItemRun.from_dict(runs_type_0_item_data)

                    runs_type_0.append(runs_type_0_item)

                return runs_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[WorkItemRun] | None, data)

        runs = _parse_runs(d.pop("runs"))

        def _parse_sessions(data: object) -> list[Session] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                sessions_type_0 = []
                _sessions_type_0 = data
                for sessions_type_0_item_data in _sessions_type_0:
                    sessions_type_0_item = Session.from_dict(sessions_type_0_item_data)

                    sessions_type_0.append(sessions_type_0_item)

                return sessions_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[Session] | None, data)

        sessions = _parse_sessions(d.pop("sessions"))

        def _parse_work_items(data: object) -> list[WorkItem] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                work_items_type_0 = []
                _work_items_type_0 = data
                for work_items_type_0_item_data in _work_items_type_0:
                    work_items_type_0_item = WorkItem.from_dict(
                        work_items_type_0_item_data
                    )

                    work_items_type_0.append(work_items_type_0_item)

                return work_items_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[WorkItem] | None, data)

        work_items = _parse_work_items(d.pop("workItems"))

        project_detail = cls(
            project=project,
            runs=runs,
            sessions=sessions,
            work_items=work_items,
        )

        project_detail.additional_properties = d
        return project_detail

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
