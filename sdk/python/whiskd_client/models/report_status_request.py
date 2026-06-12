from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ReportStatusRequest")


@_attrs_define
class ReportStatusRequest:
    """
    Attributes:
        kind (str):
        message (str):
        actor (str | Unset):
        project_id (str | Unset):
        pty_id (str | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
        work_item_id (str | Unset):
    """

    kind: str
    message: str
    actor: str | Unset = UNSET
    project_id: str | Unset = UNSET
    pty_id: str | Unset = UNSET
    run_id: str | Unset = UNSET
    session_id: str | Unset = UNSET
    work_item_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        kind = self.kind

        message = self.message

        actor = self.actor

        project_id = self.project_id

        pty_id = self.pty_id

        run_id = self.run_id

        session_id = self.session_id

        work_item_id = self.work_item_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "kind": kind,
                "message": message,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if project_id is not UNSET:
            field_dict["projectId"] = project_id
        if pty_id is not UNSET:
            field_dict["ptyId"] = pty_id
        if run_id is not UNSET:
            field_dict["runId"] = run_id
        if session_id is not UNSET:
            field_dict["sessionId"] = session_id
        if work_item_id is not UNSET:
            field_dict["workItemId"] = work_item_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = d.pop("kind")

        message = d.pop("message")

        actor = d.pop("actor", UNSET)

        project_id = d.pop("projectId", UNSET)

        pty_id = d.pop("ptyId", UNSET)

        run_id = d.pop("runId", UNSET)

        session_id = d.pop("sessionId", UNSET)

        work_item_id = d.pop("workItemId", UNSET)

        report_status_request = cls(
            kind=kind,
            message=message,
            actor=actor,
            project_id=project_id,
            pty_id=pty_id,
            run_id=run_id,
            session_id=session_id,
            work_item_id=work_item_id,
        )

        report_status_request.additional_properties = d
        return report_status_request

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
