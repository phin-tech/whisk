from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.run_event import RunEvent
    from ..models.work_item_run_metadata import WorkItemRunMetadata


T = TypeVar("T", bound="WorkItemRun")


@_attrs_define
class WorkItemRun:
    """
    Attributes:
        created_at (datetime.datetime):
        history (list[RunEvent] | None):
        id (str):
        preset (str):
        project_id (str):
        prompt_snapshot (str):
        prompt_template_id (str):
        status (str):
        updated_at (datetime.datetime):
        work_item_id (str):
        completed_at (datetime.datetime | None | Unset):
        metadata (WorkItemRunMetadata | Unset):
        pty_id (str | Unset):
        session_id (str | Unset):
    """

    created_at: datetime.datetime
    history: list[RunEvent] | None
    id: str
    preset: str
    project_id: str
    prompt_snapshot: str
    prompt_template_id: str
    status: str
    updated_at: datetime.datetime
    work_item_id: str
    completed_at: datetime.datetime | None | Unset = UNSET
    metadata: WorkItemRunMetadata | Unset = UNSET
    pty_id: str | Unset = UNSET
    session_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        history: list[dict[str, Any]] | None
        if isinstance(self.history, list):
            history = []
            for history_type_0_item_data in self.history:
                history_type_0_item = history_type_0_item_data.to_dict()
                history.append(history_type_0_item)

        else:
            history = self.history

        id = self.id

        preset = self.preset

        project_id = self.project_id

        prompt_snapshot = self.prompt_snapshot

        prompt_template_id = self.prompt_template_id

        status = self.status

        updated_at = self.updated_at.isoformat()

        work_item_id = self.work_item_id

        completed_at: None | str | Unset
        if isinstance(self.completed_at, Unset):
            completed_at = UNSET
        elif isinstance(self.completed_at, datetime.datetime):
            completed_at = self.completed_at.isoformat()
        else:
            completed_at = self.completed_at

        metadata: dict[str, Any] | Unset = UNSET
        if not isinstance(self.metadata, Unset):
            metadata = self.metadata.to_dict()

        pty_id = self.pty_id

        session_id = self.session_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "history": history,
                "id": id,
                "preset": preset,
                "projectId": project_id,
                "promptSnapshot": prompt_snapshot,
                "promptTemplateId": prompt_template_id,
                "status": status,
                "updatedAt": updated_at,
                "workItemId": work_item_id,
            }
        )
        if completed_at is not UNSET:
            field_dict["completedAt"] = completed_at
        if metadata is not UNSET:
            field_dict["metadata"] = metadata
        if pty_id is not UNSET:
            field_dict["ptyId"] = pty_id
        if session_id is not UNSET:
            field_dict["sessionId"] = session_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.run_event import RunEvent
        from ..models.work_item_run_metadata import WorkItemRunMetadata

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        def _parse_history(data: object) -> list[RunEvent] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                history_type_0 = []
                _history_type_0 = data
                for history_type_0_item_data in _history_type_0:
                    history_type_0_item = RunEvent.from_dict(history_type_0_item_data)

                    history_type_0.append(history_type_0_item)

                return history_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[RunEvent] | None, data)

        history = _parse_history(d.pop("history"))

        id = d.pop("id")

        preset = d.pop("preset")

        project_id = d.pop("projectId")

        prompt_snapshot = d.pop("promptSnapshot")

        prompt_template_id = d.pop("promptTemplateId")

        status = d.pop("status")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        work_item_id = d.pop("workItemId")

        def _parse_completed_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                completed_at_type_0 = datetime.datetime.fromisoformat(data)

                return completed_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        completed_at = _parse_completed_at(d.pop("completedAt", UNSET))

        _metadata = d.pop("metadata", UNSET)
        metadata: WorkItemRunMetadata | Unset
        if isinstance(_metadata, Unset):
            metadata = UNSET
        else:
            metadata = WorkItemRunMetadata.from_dict(_metadata)

        pty_id = d.pop("ptyId", UNSET)

        session_id = d.pop("sessionId", UNSET)

        work_item_run = cls(
            created_at=created_at,
            history=history,
            id=id,
            preset=preset,
            project_id=project_id,
            prompt_snapshot=prompt_snapshot,
            prompt_template_id=prompt_template_id,
            status=status,
            updated_at=updated_at,
            work_item_id=work_item_id,
            completed_at=completed_at,
            metadata=metadata,
            pty_id=pty_id,
            session_id=session_id,
        )

        work_item_run.additional_properties = d
        return work_item_run

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
