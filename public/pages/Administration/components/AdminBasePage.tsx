import "./AdminBasePage.scss"

import React from "react"
import { Header, PageTitle } from "@fider/components"
import { SideMenu, SideMenuToggler } from "./SideMenu"
import { HStack } from "@fider/components/layout"

interface AdminPageContainerProps {
  id: string
  name: string
  title: string
  subtitle: string
  bare?: boolean
  children: React.ReactNode
}

export const AdminPageContainer = (props: AdminPageContainerProps) => {
  return (
    <>
      <Header />
      <div id={props.id} className="page container">
        <HStack justify="between" className="c-admin-head">
          <div>
            <span className="c-admin-head__eyebrow">Admin</span>
            <PageTitle title={props.title} subtitle={props.subtitle} />
          </div>
          <SideMenuToggler />
        </HStack>

        <div className="c-admin-basepage">
          <SideMenu activeItem={props.name} />
          <div className={props.bare ? "c-admin-content c-admin-content--bare" : "c-admin-content"}>
            {!props.bare && (
              <div className="c-admin-content__head">
                <span className="c-admin-content__eyebrow">{props.title}</span>
                <span className="c-admin-content__hint">{props.subtitle}</span>
              </div>
            )}
            {props.children}
          </div>
        </div>
      </div>
    </>
  )
}

export abstract class AdminBasePage<P, S> extends React.Component<P, S> {
  public abstract id: string
  public abstract name: string
  public abstract title: string
  public abstract subtitle: string
  // Pages that render their own panel cards (e.g. scorecard admin) set this
  // to opt out of the shared content card.
  public bare = false
  public abstract content(): JSX.Element

  public render() {
    return (
      <AdminPageContainer id={this.id} name={this.name} title={this.title} subtitle={this.subtitle} bare={this.bare}>
        {this.content()}
      </AdminPageContainer>
    )
  }
}
