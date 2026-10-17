import React from 'react'

interface StepProps {
    number: number
    title: string
    description: React.ReactNode
    children: React.ReactNode
    isActive?: boolean
    isCompleted?: boolean
}

export function Step({ number, title, description, children, isActive = true, isCompleted = false }: StepProps) {
    return (
        <div className={`step-card ${isActive ? 'active' : ''} ${isCompleted ? 'completed' : ''}`}>
            <div className="step-header">
                <div className="step-number">{number}</div>
                <h3>{title}</h3>
            </div>
            <div className="step-content">
                <div className="step-description">{description}</div>
                <div className="step-action">{children}</div>
            </div>
        </div>
    )
}
