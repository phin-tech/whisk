from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.agent_status import AgentStatus


T = TypeVar("T", bound="PTYInfo")


@_attrs_define
class PTYInfo:
    """
    Attributes:
        cols (int):
        id (str):
        origin_pane_id (str):
        origin_window_id (str):
        pane_id (str):
        rows (int):
        running (bool):
        session_id (str):
        status (str):
        window_id (str):
        working_dir (str):
        agent_status (AgentStatus | None | Unset):
        exit_code (int | None | Unset):
        terminal_working_directory (str | Unset):
        title (str | Unset):
    """

    cols: int
    id: str
    origin_pane_id: str
    origin_window_id: str
    pane_id: str
    rows: int
    running: bool
    session_id: str
    status: str
    window_id: str
    working_dir: str
    agent_status: AgentStatus | None | Unset = UNSET
    exit_code: int | None | Unset = UNSET
    terminal_working_directory: str | Unset = UNSET
    title: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.agent_status import AgentStatus

        cols = self.cols

        id = self.id

        origin_pane_id = self.origin_pane_id

        origin_window_id = self.origin_window_id

        pane_id = self.pane_id

        rows = self.rows

        running = self.running

        session_id = self.session_id

        status = self.status

        window_id = self.window_id

        working_dir = self.working_dir

        agent_status: dict[str, Any] | None | Unset
        if isinstance(self.agent_status, Unset):
            agent_status = UNSET
        elif isinstance(self.agent_status, AgentStatus):
            agent_status = self.agent_status.to_dict()
        else:
            agent_status = self.agent_status

        exit_code: int | None | Unset
        if isinstance(self.exit_code, Unset):
            exit_code = UNSET
        else:
            exit_code = self.exit_code

        terminal_working_directory = self.terminal_working_directory

        title = self.title

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "cols": cols,
                "id": id,
                "originPaneId": origin_pane_id,
                "originWindowId": origin_window_id,
                "paneId": pane_id,
                "rows": rows,
                "running": running,
                "sessionId": session_id,
                "status": status,
                "windowId": window_id,
                "workingDir": working_dir,
            }
        )
        if agent_status is not UNSET:
            field_dict["agentStatus"] = agent_status
        if exit_code is not UNSET:
            field_dict["exitCode"] = exit_code
        if terminal_working_directory is not UNSET:
            field_dict["terminalWorkingDirectory"] = terminal_working_directory
        if title is not UNSET:
            field_dict["title"] = title

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.agent_status import AgentStatus

        d = dict(src_dict)
        cols = d.pop("cols")

        id = d.pop("id")

        origin_pane_id = d.pop("originPaneId")

        origin_window_id = d.pop("originWindowId")

        pane_id = d.pop("paneId")

        rows = d.pop("rows")

        running = d.pop("running")

        session_id = d.pop("sessionId")

        status = d.pop("status")

        window_id = d.pop("windowId")

        working_dir = d.pop("workingDir")

        def _parse_agent_status(data: object) -> AgentStatus | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                agent_status_type_1 = AgentStatus.from_dict(data)

                return agent_status_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(AgentStatus | None | Unset, data)

        agent_status = _parse_agent_status(d.pop("agentStatus", UNSET))

        def _parse_exit_code(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        exit_code = _parse_exit_code(d.pop("exitCode", UNSET))

        terminal_working_directory = d.pop("terminalWorkingDirectory", UNSET)

        title = d.pop("title", UNSET)

        pty_info = cls(
            cols=cols,
            id=id,
            origin_pane_id=origin_pane_id,
            origin_window_id=origin_window_id,
            pane_id=pane_id,
            rows=rows,
            running=running,
            session_id=session_id,
            status=status,
            window_id=window_id,
            working_dir=working_dir,
            agent_status=agent_status,
            exit_code=exit_code,
            terminal_working_directory=terminal_working_directory,
            title=title,
        )

        pty_info.additional_properties = d
        return pty_info

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
